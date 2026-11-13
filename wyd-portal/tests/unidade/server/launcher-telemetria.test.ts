import { describe, expect, it } from "vitest";

import {
  esquemaEvento,
  registrar,
  resumo,
  resumoTabela,
} from "@/server/launcher-telemetria";

/**
 * Contrato da telemetria: o que o launcher pode mandar, o que é recusado e como
 * o agregado se comporta. Nenhum teste usa `reset` de módulo — cada um escolhe
 * uma `versao` própria e lê o delta, então a ordem de execução não importa.
 */

const base = {
  versao: 7001,
  build_cliente: 7662,
  operacao: "patch" as const,
  ok: true,
  duracao_ms: 1200,
  arquivos: 3,
  falhas: [],
};

describe("esquema da telemetria", () => {
  it("aceita evento mínimo e preenche opcionais", () => {
    const lido = esquemaEvento.safeParse({
      versao: 1,
      build_cliente: 7662,
      operacao: "check",
      ok: false,
      duracao_ms: 0,
    });
    expect(lido.success).toBe(true);
    if (lido.success) {
      expect(lido.data.arquivos).toBe(0);
      expect(lido.data.falhas).toEqual([]);
    }
  });

  it("recusa campo extra (PII ou invenção do cliente)", () => {
    const lido = esquemaEvento.safeParse({ ...base, login: "jogador" });
    expect(lido.success).toBe(false);
  });

  it("recusa caminho absoluto na falha (só nome de arquivo sai da máquina)", () => {
    const lido = esquemaEvento.safeParse({
      ...base,
      falhas: [{ arquivo: "C:\\jogo\\data\\jogo.dat", motivo: "hash" }],
    });
    expect(lido.success).toBe(false);
  });

  it("recusa duração acima do teto e mais de 10 falhas", () => {
    expect(esquemaEvento.safeParse({ ...base, duracao_ms: 600_001 }).success).toBe(false);
    const falhas = Array.from({ length: 11 }, (_, i) => ({ arquivo: `a${i}.dat`, motivo: "io" }));
    expect(esquemaEvento.safeParse({ ...base, falhas }).success).toBe(false);
  });
});

describe("agregação", () => {
  it("soma eventos, falhas e duração por versão + operação", () => {
    const versao = 7101;
    const antes = resumo().chaves.find((c) => c.versao === versao)?.eventos ?? 0;

    registrar(
      {
        ...base,
        versao,
        duracao_ms: 100,
        falhas: [{ arquivo: "jogo.dat", motivo: "hash" }],
      },
      new Date("2026-11-11T12:00:00Z"),
    );
    registrar({ ...base, versao, duracao_ms: 300 }, new Date("2026-11-11T12:00:05Z"));

    const chave = resumo().chaves.find((c) => c.versao === versao);
    expect(chave).toBeDefined();
    expect(chave!.eventos - antes).toBe(2);
    expect(chave!.falhas).toBe(1);
    expect(chave!.duracaoTotalMs).toBe(400);
    expect(chave!.duracaoMaxMs).toBe(300);
    expect(chave!.ultimaEm).toBe("2026-11-11T12:00:05.000Z");
  });

  it("separa operações diferentes da mesma versão", () => {
    const versao = 7102;
    registrar({ ...base, versao, operacao: "patch" });
    registrar({ ...base, versao, operacao: "launch" });

    const daVersao = resumo().chaves.filter((c) => c.versao === versao);
    expect(daVersao.map((c) => c.operacao).sort()).toEqual(["launch", "patch"]);
  });

  it("o painel calcula média e taxa de falha e ordena por versão", () => {
    const versao = 7200;
    registrar(
      { ...base, versao, duracao_ms: 100, falhas: [{ arquivo: "a.dat", motivo: "hash" }] },
      new Date("2026-11-12T10:00:00Z"),
    );
    registrar({ ...base, versao, duracao_ms: 300 }, new Date("2026-11-12T10:00:05Z"));

    const linhas = resumoTabela();
    const linha = linhas.find((item) => item.versao === versao)!;
    expect(linha.eventos).toBe(2);
    expect(linha.falhas).toBe(1);
    expect(linha.mediaMs).toBe(200);
    expect(linha.taxaFalha).toBe(0.5);

    const versoes = linhas.map((item) => item.versao);
    expect(versoes).toEqual([...versoes].sort((a, b) => b - a));
  });

  it("a foto é cópia: alterar o retorno não mexe no agregado", () => {
    const versao = 7103;
    registrar({ ...base, versao });
    const foto = resumo();
    const chave = foto.chaves.find((c) => c.versao === versao)!;
    chave.eventos = 999;
    expect(resumo().chaves.find((c) => c.versao === versao)!.eventos).toBe(1);
  });
});
