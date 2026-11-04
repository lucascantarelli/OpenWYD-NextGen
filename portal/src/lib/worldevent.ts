/**
 * Configuração do evento global de mundo (issue 28 / `WorldEventAdminService`).
 *
 * O contrato é `web.proto:908` e o `SetWorldEventConfig` **substitui o config
 * inteiro** — não existe patch parcial. O tmserver relê por polling (15 s) e
 * aplica no loop, então uma alteração salva chega ao mundo sem restart.
 *
 * A validação aqui espelha `webserver/internal/worldevent.validConfig`; ela roda
 * no BFF (autoridade) e no formulário (feedback imediato). Divergir das duas
 * produz um "salvar" que o servidor recusa sem explicação.
 */
import { fmtCoin } from "./format";

export interface WorldEventConfig {
  enabled: boolean;
  /** Item entregue no drop global (`ItemList.csv`). */
  item_index: number;
  /** Divisor do roll: `1` = sempre que elegível. */
  rate: number;
  start_index: number;
  current_index: number;
  end_index: number;
  /** Serializa o item com os efeitos 62/63 e o aleatório 59. */
  indexed: boolean;
  notice_enabled: boolean;
  double_exp_enabled: boolean;
  newbie_event_enabled: boolean;
}

/** Limite do backend (`maxWorldEventItemIndex`); acima disso é INVALID. */
export const ITEM_INDEX_MAX = 32767;

/**
 * Teto de UX para o divisor. O backend aceita qualquer inteiro > 0, mas um
 * divisor acima disso é "nunca cai" na prática e só esconde erro de digitação.
 */
export const RATE_MAX = 10000;

/** Config que o web-api devolve quando nada foi editado ainda. */
export function emptyConfig(): WorldEventConfig {
  return {
    enabled: false,
    item_index: 0,
    rate: 0,
    start_index: 0,
    current_index: 0,
    end_index: 0,
    indexed: false,
    notice_enabled: false,
    double_exp_enabled: false,
    newbie_event_enabled: false,
  };
}

export type WorldEventStatus = "off" | "incomplete" | "active" | "exhausted";

export interface StatusInfo {
  status: WorldEventStatus;
  label: string;
  tone: "neutral" | "ok" | "warn" | "danger";
  hint: string;
}

/**
 * Estado do **drop global** (não dos multiplicadores). Um evento `enabled` com
 * números zerados é config inválida, mas pode existir no banco de antes da
 * validação — a tela precisa dizer isso em vez de mostrar "ativo".
 */
export function worldEventStatus(cfg: WorldEventConfig): StatusInfo {
  if (!cfg.enabled) {
    return {
      status: "off",
      label: "desligado",
      tone: "neutral",
      hint: "o drop global não rola; nenhum item de evento é entregue.",
    };
  }
  if (cfg.item_index <= 0 || cfg.rate <= 0 || cfg.start_index <= 0) {
    return {
      status: "incomplete",
      label: "config incompleta",
      tone: "danger",
      hint: "evento ligado sem item, divisor ou índice inicial: o drop é ignorado até corrigir.",
    };
  }
  if (cfg.current_index >= cfg.end_index) {
    return {
      status: "exhausted",
      label: "esgotado",
      tone: "warn",
      hint: "o progresso chegou ao fim da faixa: o drop parou de rolar.",
    };
  }
  return {
    status: "active",
    label: "ativo",
    tone: "ok",
    hint: "o drop rola a cada kill elegível e avança o progresso.",
  };
}

/** `rate` é divisor, não porcentagem: 1 = sempre, 10 = 1 em cada 10. */
export function rateLabel(rate: number): string {
  const n = Math.trunc(rate);
  if (!Number.isFinite(n) || n <= 0) return "sem chance (divisor 0)";
  if (n === 1) return "sempre que elegível";
  return `1 em cada ${fmtCoin(n)} kills elegíveis`;
}

/** Quantos drops ainda cabem na faixa antes do evento esgotar. */
export function remainingDrops(cfg: WorldEventConfig): number {
  if (cfg.end_index <= cfg.current_index) return 0;
  return cfg.end_index - cfg.current_index;
}

export type ConfigErrors = Partial<Record<keyof WorldEventConfig, string>>;

/**
 * Espelho de `validConfig` do web-api: negativos nunca valem; com o evento
 * desligado os números podem ficar zerados; ligado exige faixa coerente.
 */
export function validateWorldEvent(cfg: WorldEventConfig): ConfigErrors {
  const errors: ConfigErrors = {};

  if (cfg.item_index < 0) errors.item_index = "não pode ser negativo";
  else if (cfg.item_index > ITEM_INDEX_MAX)
    errors.item_index = `o limite do servidor é ${fmtCoin(ITEM_INDEX_MAX)}`;
  if (cfg.rate < 0) errors.rate = "não pode ser negativo";
  if (cfg.start_index < 0) errors.start_index = "não pode ser negativo";
  if (cfg.current_index < 0) errors.current_index = "não pode ser negativo";
  if (cfg.end_index < 0) errors.end_index = "não pode ser negativo";

  if (!cfg.enabled) return errors;

  if (cfg.item_index <= 0) errors.item_index = "escolha o item do evento";
  if (cfg.rate <= 0) errors.rate = "o divisor precisa ser maior que zero";
  if (cfg.start_index <= 0) errors.start_index = "precisa ser maior que zero";
  if (cfg.end_index <= cfg.start_index)
    errors.end_index = "precisa ser maior que o índice inicial";
  if (cfg.current_index < cfg.start_index)
    errors.current_index = "não pode ser menor que o índice inicial";
  else if (cfg.current_index > cfg.end_index)
    errors.current_index = "não pode passar do índice final";

  return errors;
}

/** Compara o rascunho com o que o servidor confirmou (para "não salvo"). */
export function configChanged(a: WorldEventConfig, b: WorldEventConfig): boolean {
  return (Object.keys(a) as (keyof WorldEventConfig)[]).some((key) => a[key] !== b[key]);
}
