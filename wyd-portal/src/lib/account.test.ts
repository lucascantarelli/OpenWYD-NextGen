import { describe, expect, it } from "vitest";

import {
  actionLabel,
  blockedLabel,
  emailError,
  fmtUnix,
  nameError,
  passwordError,
  reasonError,
  roleLabel,
  roleTone,
} from "./account";

describe("conta: papéis", () => {
  it("rotula os três papéis e cai no valor cru quando desconhecido", () => {
    expect(roleLabel("player")).toBe("Jogador");
    expect(roleLabel("moderator")).toBe("Moderador");
    expect(roleLabel("admin")).toBe("Admin");
    expect(roleLabel("root")).toBe("root");
  });

  it("dá tom de destaque para admin e moderador", () => {
    expect(roleTone("admin")).toBe("warn");
    expect(roleTone("moderator")).toBe("accent");
    expect(roleTone("player")).toBe("neutral");
  });
});

describe("conta: validação de criação", () => {
  it("aceita login de 4 a 12 letras/números e recusa o resto", () => {
    expect(nameError("teste")).toBeNull();
    expect(nameError("abc")).not.toBeNull();
    expect(nameError("a".repeat(13))).not.toBeNull();
    expect(nameError("com espaço")).not.toBeNull();
    expect(nameError("joão")).not.toBeNull();
    expect(nameError("  Teste  ")).toBeNull();
  });

  it("exige senha de 4 e e-mail só quando preenchido", () => {
    expect(passwordError("123")).not.toBeNull();
    expect(passwordError("1234")).toBeNull();
    expect(emailError("")).toBeNull();
    expect(emailError("nao-email")).not.toBeNull();
    expect(emailError("a@b.com")).toBeNull();
  });
});

describe("conta: motivo", () => {
  it("exige 10 caracteres e ignora espaço nas pontas", () => {
    expect(reasonError("curto")).not.toBeNull();
    expect(reasonError("          ")).not.toBeNull();
    expect(reasonError("promoção para moderador")).toBeNull();
  });
});

describe("conta: apresentação", () => {
  it("rotula filtro de bloqueio", () => {
    expect(blockedLabel(-1)).toBe("todas");
    expect(blockedLabel(0)).toBe("ativas");
    expect(blockedLabel(1)).toBe("bloqueadas");
  });

  it("rotula ações de auditoria", () => {
    expect(actionLabel("set_role")).toBe("papel alterado");
    expect(actionLabel("set_blocked")).toBe("bloqueio alterado");
    expect(actionLabel("create")).toBe("conta criada");
    expect(actionLabel("x")).toBe("x");
  });

  it("formata data em São Paulo e trata 0 como ausente", () => {
    expect(fmtUnix(0)).toBe("—");
    expect(fmtUnix(1_700_000_000)).toMatch(/2023/);
  });
});
