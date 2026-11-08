//! JSON do subconjunto do contrato admin (ADR-0008), sem dependência externa.
//!
//! Cobre objeto/array/string/número/bool/null — o que o contrato usa. Limites
//! explícitos (`MAX_DEPTH`) e nenhum `unsafe`: entrada de rede não causa `panic!`.

use core::fmt;

/// Profundidade máxima de aninhamento (entrada hostil não estoura a pilha).
pub const MAX_DEPTH: usize = 32;

/// Valor JSON do contrato.
#[derive(Debug, Clone, PartialEq)]
pub enum Json {
    Null,
    Bool(bool),
    Num(f64),
    Str(String),
    Arr(Vec<Json>),
    Obj(Vec<(String, Json)>),
}

/// Falha de parse (mensagem curta, sem ecoar o corpo recebido).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct JsonError {
    pub reason: &'static str,
}

impl fmt::Display for JsonError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.reason)
    }
}

impl std::error::Error for JsonError {}

impl Json {
    /// Parseia um documento completo; sobra de bytes é erro (nada de lixo no fim).
    pub fn parse(input: &str) -> Result<Self, JsonError> {
        let mut parser = Parser {
            bytes: input.as_bytes(),
            pos: 0,
            depth: 0,
        };
        let value = parser.value()?;
        parser.skip_ws();
        if parser.pos != parser.bytes.len() {
            return Err(err("conteúdo depois do JSON"));
        }
        Ok(value)
    }

    /// Campo de objeto (case-sensitive, como no contrato).
    #[must_use]
    pub fn get(&self, key: &str) -> Option<&Self> {
        match self {
            Self::Obj(fields) => fields.iter().find(|(k, _)| k == key).map(|(_, v)| v),
            _ => None,
        }
    }

    #[must_use]
    pub fn as_str(&self) -> Option<&str> {
        match self {
            Self::Str(s) => Some(s),
            _ => None,
        }
    }

    #[must_use]
    pub fn as_bool(&self) -> Option<bool> {
        match self {
            Self::Bool(b) => Some(*b),
            _ => None,
        }
    }

    /// Inteiro exato (fração ou fora de `i64` não é aceito como inteiro).
    #[must_use]
    pub fn as_i64(&self) -> Option<i64> {
        match self {
            Self::Num(n) if n.fract() == 0.0 && *n >= i64::MIN as f64 && *n <= i64::MAX as f64 => {
                Some(*n as i64)
            }
            _ => None,
        }
    }

    /// Renderiza o valor como JSON compacto.
    #[must_use]
    pub fn render(&self) -> String {
        let mut out = String::new();
        self.write(&mut out);
        out
    }

    fn write(&self, out: &mut String) {
        match self {
            Self::Null => out.push_str("null"),
            Self::Bool(true) => out.push_str("true"),
            Self::Bool(false) => out.push_str("false"),
            Self::Num(n) => write_number(out, *n),
            Self::Str(s) => push_quoted(out, s),
            Self::Arr(items) => {
                out.push('[');
                for (i, item) in items.iter().enumerate() {
                    if i > 0 {
                        out.push(',');
                    }
                    item.write(out);
                }
                out.push(']');
            }
            Self::Obj(fields) => {
                out.push('{');
                for (i, (key, value)) in fields.iter().enumerate() {
                    if i > 0 {
                        out.push(',');
                    }
                    push_quoted(out, key);
                    out.push(':');
                    value.write(out);
                }
                out.push('}');
            }
        }
    }
}

impl fmt::Display for Json {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.render())
    }
}

/// String já escapada e entre aspas (para compor o envelope manualmente).
#[must_use]
pub fn quote(input: &str) -> String {
    let mut out = String::with_capacity(input.len() + 2);
    push_quoted(&mut out, input);
    out
}

fn push_quoted(out: &mut String, input: &str) {
    out.push('"');
    for ch in input.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => {
                let code = c as u32;
                out.push_str(&format!("\\u{code:04x}"));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

fn write_number(out: &mut String, n: f64) {
    if !n.is_finite() {
        // JSON não tem NaN/Infinity; o contrato nunca os produz.
        out.push_str("null");
    } else if n.fract() == 0.0 && n >= i64::MIN as f64 && n <= i64::MAX as f64 {
        let whole = n as i64;
        out.push_str(&format!("{whole}"));
    } else {
        out.push_str(&format!("{n}"));
    }
}

const fn err(reason: &'static str) -> JsonError {
    JsonError { reason }
}

struct Parser<'a> {
    bytes: &'a [u8],
    pos: usize,
    depth: usize,
}

impl Parser<'_> {
    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    fn bump(&mut self) -> Option<u8> {
        let byte = self.peek()?;
        self.pos += 1;
        Some(byte)
    }

    fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\n' | b'\r' | b'\t')) {
            self.pos += 1;
        }
    }

    fn expect(&mut self, byte: u8) -> Result<(), JsonError> {
        if self.bump() == Some(byte) {
            Ok(())
        } else {
            Err(err("estrutura JSON inválida"))
        }
    }

    fn value(&mut self) -> Result<Json, JsonError> {
        self.skip_ws();
        match self.peek() {
            Some(b'{') => self.object(),
            Some(b'[') => self.array(),
            Some(b'"') => Ok(Json::Str(self.string()?)),
            Some(b't') => self.literal(b"true", Json::Bool(true)),
            Some(b'f') => self.literal(b"false", Json::Bool(false)),
            Some(b'n') => self.literal(b"null", Json::Null),
            Some(b'-' | b'0'..=b'9') => self.number(),
            Some(_) => Err(err("valor JSON inválido")),
            None => Err(err("JSON vazio")),
        }
    }

    fn literal(&mut self, word: &[u8], value: Json) -> Result<Json, JsonError> {
        let end = self.pos.saturating_add(word.len());
        if self.bytes.get(self.pos..end) == Some(word) {
            self.pos = end;
            Ok(value)
        } else {
            Err(err("literal JSON inválido"))
        }
    }

    fn enter(&mut self) -> Result<(), JsonError> {
        self.depth += 1;
        if self.depth > MAX_DEPTH {
            return Err(err("JSON aninhado demais"));
        }
        Ok(())
    }

    fn object(&mut self) -> Result<Json, JsonError> {
        self.enter()?;
        self.expect(b'{')?;
        self.skip_ws();
        let mut fields = Vec::new();
        if self.peek() == Some(b'}') {
            self.pos += 1;
            self.depth -= 1;
            return Ok(Json::Obj(fields));
        }
        loop {
            self.skip_ws();
            let key = self.string()?;
            self.skip_ws();
            self.expect(b':')?;
            let value = self.value()?;
            fields.push((key, value));
            self.skip_ws();
            match self.bump() {
                Some(b',') => continue,
                Some(b'}') => break,
                _ => return Err(err("objeto JSON sem fechamento válido")),
            }
        }
        self.depth -= 1;
        Ok(Json::Obj(fields))
    }

    fn array(&mut self) -> Result<Json, JsonError> {
        self.enter()?;
        self.expect(b'[')?;
        self.skip_ws();
        let mut items = Vec::new();
        if self.peek() == Some(b']') {
            self.pos += 1;
            self.depth -= 1;
            return Ok(Json::Arr(items));
        }
        loop {
            items.push(self.value()?);
            self.skip_ws();
            match self.bump() {
                Some(b',') => continue,
                Some(b']') => break,
                _ => return Err(err("array JSON sem fechamento válido")),
            }
        }
        self.depth -= 1;
        Ok(Json::Arr(items))
    }

    fn string(&mut self) -> Result<String, JsonError> {
        self.expect(b'"')?;
        let mut out: Vec<u8> = Vec::new();
        loop {
            match self.bump() {
                None => return Err(err("string JSON sem fechamento")),
                Some(b'"') => break,
                Some(b'\\') => self.escape(&mut out)?,
                Some(byte) if byte < 0x20 => return Err(err("controle cru na string")),
                Some(byte) => out.push(byte),
            }
        }
        String::from_utf8(out).map_err(|_| err("string JSON não é UTF-8"))
    }

    fn escape(&mut self, out: &mut Vec<u8>) -> Result<(), JsonError> {
        match self.bump() {
            Some(b'"') => out.push(b'"'),
            Some(b'\\') => out.push(b'\\'),
            Some(b'/') => out.push(b'/'),
            Some(b'b') => out.push(0x08),
            Some(b'f') => out.push(0x0c),
            Some(b'n') => out.push(b'\n'),
            Some(b'r') => out.push(b'\r'),
            Some(b't') => out.push(b'\t'),
            Some(b'u') => {
                let first = self.hex4()?;
                let code = if (0xD800..=0xDBFF).contains(&first) {
                    self.expect(b'\\')?;
                    self.expect(b'u')?;
                    let second = self.hex4()?;
                    if !(0xDC00..=0xDFFF).contains(&second) {
                        return Err(err("surrogate JSON inválido"));
                    }
                    0x1_0000 + ((first - 0xD800) << 10) + (second - 0xDC00)
                } else {
                    first
                };
                let ch = char::from_u32(code).ok_or(err("escape unicode inválido"))?;
                let mut buf = [0u8; 4];
                out.extend_from_slice(ch.encode_utf8(&mut buf).as_bytes());
            }
            _ => return Err(err("escape JSON inválido")),
        }
        Ok(())
    }

    fn hex4(&mut self) -> Result<u32, JsonError> {
        let mut code = 0u32;
        for _ in 0..4 {
            let digit = match self.bump() {
                Some(b @ b'0'..=b'9') => u32::from(b - b'0'),
                Some(b @ b'a'..=b'f') => u32::from(b - b'a') + 10,
                Some(b @ b'A'..=b'F') => u32::from(b - b'A') + 10,
                _ => return Err(err("escape unicode inválido")),
            };
            code = code * 16 + digit;
        }
        Ok(code)
    }

    fn number(&mut self) -> Result<Json, JsonError> {
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.pos += 1;
        }
        let digits_start = self.pos;
        while matches!(self.peek(), Some(b'0'..=b'9')) {
            self.pos += 1;
        }
        if self.pos == digits_start {
            return Err(err("número JSON inválido"));
        }
        if let Some(digits) = self.bytes.get(digits_start..self.pos) {
            // JSON não aceita zero à esquerda ("01"); fração/expoente vêm depois.
            if digits.len() > 1 && digits.first() == Some(&b'0') {
                return Err(err("número JSON inválido"));
            }
        }
        if self.peek() == Some(b'.') {
            self.pos += 1;
            let frac_start = self.pos;
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
            if self.pos == frac_start {
                return Err(err("número JSON inválido"));
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            let exp_start = self.pos;
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
            if self.pos == exp_start {
                return Err(err("número JSON inválido"));
            }
        }
        let slice = self
            .bytes
            .get(start..self.pos)
            .ok_or(err("número JSON inválido"))?;
        let text = std::str::from_utf8(slice).map_err(|_| err("número JSON inválido"))?;
        let parsed = text
            .parse::<f64>()
            .map_err(|_| err("número JSON inválido"))?;
        Ok(Json::Num(parsed))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_contract_shapes() {
        let value = Json::parse(r#"{"login":"alvo","role":"player","n":7}"#).expect("objeto");
        assert_eq!(value.get("login").and_then(Json::as_str), Some("alvo"));
        assert_eq!(value.get("n").and_then(Json::as_i64), Some(7));
        let nested = Json::parse(r#"{"kind":{"type":"gold","amount":100}}"#).expect("aninhado");
        assert_eq!(
            nested
                .get("kind")
                .and_then(|k| k.get("type"))
                .and_then(Json::as_str),
            Some("gold")
        );
    }

    #[test]
    fn roundtrips_escapes() {
        let value = Json::Str("a\"b\\c\n\u{1}".to_string());
        let rendered = value.render();
        assert_eq!(Json::parse(&rendered).expect("volta"), value);
        assert_eq!(quote("x\"y"), r#""x\"y""#);
    }

    #[test]
    fn rejects_trailing_garbage_and_broken_shapes() {
        assert!(Json::parse(r#"{"a":1} lixo"#).is_err());
        assert!(Json::parse("{").is_err());
        assert!(Json::parse("[1,]").is_err());
        assert!(Json::parse("01").is_err());
    }

    #[test]
    fn rejects_deep_nesting_without_panicking() {
        let deep = "[".repeat(MAX_DEPTH + 1);
        assert!(Json::parse(&deep).is_err());
    }

    #[test]
    fn as_i64_rejects_fraction() {
        assert_eq!(Json::Num(1.5).as_i64(), None);
        assert_eq!(Json::Num(2.0).as_i64(), Some(2));
        assert_eq!(Json::Str("2".to_string()).as_i64(), None);
    }
}
