//! Guarda de contrato do monitoramento.
//!
//! Toda métrica Prometheus citada em `monitoring/grafana/dashboards/*.json` ou em
//! `monitoring/prometheus/alerts.yml` precisa existir na exposição real do processo
//! (servidor + gateway) ou ser de uma fonte externa declarada (`node_*` do
//! node-exporter). Todo `job="..."` precisa existir no scrape config, e todo alvo do
//! job `wyd-server` precisa do label `canal` que os painéis agrupam.
//!
//! É um teste de regressão da expansão do monitoramento: métrica renomeada, label
//! errado ou job inventado quebra aqui antes de virar gráfico vazio em produção.

use std::collections::HashSet;
use std::io::{Read, Write};
use std::net::TcpStream;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

use serde_json::Value;
use wyd_admin::metrics::GatewayMetrics;
use wyd_core::Metrics;
use wyd_server::metrics::ServiceMetrics;

fn repo_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../..")
        .canonicalize()
        .expect("raiz do workspace")
}

fn ler(raiz: &Path, relativo: &str) -> String {
    let caminho = raiz.join(relativo);
    std::fs::read_to_string(&caminho)
        .unwrap_or_else(|erro| panic!("não li {}: {erro}", caminho.display()))
}

/// Nomes de métricas publicados em um texto Prometheus 0.0.4.
fn nomes_da_exposicao(texto: &str) -> HashSet<String> {
    texto
        .lines()
        .filter(|linha| !linha.starts_with('#') && !linha.trim().is_empty())
        .filter_map(|linha| {
            let nome = linha.split(['{', ' ']).next()?.trim();
            (!nome.is_empty()).then(|| nome.to_string())
        })
        .collect()
}

/// Identificadores com cara de métrica (`wyd_*`, `process_*`) dentro de um `expr`.
fn tokens_de_metrica(expr: &str) -> Vec<String> {
    let bytes = expr.as_bytes();
    let mut tokens = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        let atual = bytes[i] as char;
        if atual.is_ascii_alphabetic() || atual == '_' {
            let inicio = i;
            while i < bytes.len() {
                let c = bytes[i] as char;
                if c.is_ascii_alphanumeric() || c == '_' || c == ':' {
                    i += 1;
                } else {
                    break;
                }
            }
            let token = &expr[inicio..i];
            if token.starts_with("wyd_") || token.starts_with("process_") {
                tokens.push(token.to_string());
            }
        } else {
            i += 1;
        }
    }
    tokens
}

/// Valores de `job="..."` citados em um `expr`.
fn jobs_citados(expr: &str) -> Vec<String> {
    let mut jobs = Vec::new();
    let mut resto = expr;
    while let Some(posicao) = resto.find("job=\"") {
        let inicio = posicao + "job=\"".len();
        let depois = &resto[inicio..];
        let Some(fim) = depois.find('"') else { break };
        jobs.push(depois[..fim].to_string());
        resto = &depois[fim..];
    }
    jobs
}

/// Queries PromQL escritas no BFF do portal (`monitoring.ts`).
///
/// O teste dos dashboards não enxerga essas consultas: elas são strings TS em
/// `await optional("...")` / `await query("...")`, às vezes na linha seguinte.
fn queries_do_portal(texto: &str) -> Vec<String> {
    let linhas: Vec<&str> = texto.lines().collect();
    let mut out = Vec::new();
    for (indice, linha) in linhas.iter().enumerate() {
        if !(linha.contains("await optional(") || linha.contains("await query(")) {
            continue;
        }
        // `await query(expr)` é o helper genérico: a chamada não traz literal.
        let resto = linha
            .split_once("await optional(")
            .or_else(|| linha.split_once("await query("))
            .map(|(_, resto)| resto)
            .unwrap_or("");
        let trecho = if resto.contains('"') || resto.contains('\'') {
            resto
        } else {
            linhas.get(indice + 1).copied().unwrap_or("")
        };
        let Some(inicio) = trecho.find(['"', '\'']) else {
            continue;
        };
        let aspas = trecho.as_bytes()[inicio] as char;
        let depois = &trecho[inicio + 1..];
        let Some(fim) = depois.find(aspas) else {
            continue;
        };
        let expr = depois[..fim].trim();
        if !expr.is_empty() {
            out.push(expr.to_string());
        }
    }
    out
}

/// Nomes de serviço declarados nos dois composes que formam a stack de dev.
fn servicos_do_compose(raiz: &Path) -> HashSet<String> {
    ["docker-compose.yml", "docker-compose.override.yml"]
        .iter()
        .flat_map(|arquivo| {
            ler(raiz, arquivo)
                .lines()
                .map(str::to_string)
                .collect::<Vec<_>>()
        })
        .filter(|linha| {
            linha.starts_with("  ") && !linha.starts_with("    ") && linha.ends_with(':')
        })
        .map(|linha| linha.trim().trim_end_matches(':').to_string())
        .collect()
}

/// Exposição real do processo: fragmento de rede + registro do motor + gauges + gateway.
fn exposicao() -> String {
    let motor = Metrics::new();
    motor.record_tick(1_200, 7);
    motor.set_heap_bytes(4 * 1024 * 1024);
    let servico = ServiceMetrics::new(2);
    servico.attach_engine(Arc::new(motor));
    let gateway = GatewayMetrics::new();
    gateway.record(200, Some(1));
    format!("{}{}", servico.render(), gateway.render())
}

#[test]
fn metricas_de_paineis_e_alertas_existem_na_exposicao() {
    let raiz = repo_root();
    let exposto = nomes_da_exposicao(&exposicao());
    for obrigatoria in [
        "wyd_ticks_total",
        "wyd_tick_duration_microseconds_bucket",
        "wyd_tick_budget_microseconds",
        "wyd_entities_active",
        "process_resident_memory_bytes",
        "wyd_memory_samples_total",
        "wyd_packets_total",
        "wyd_packets_rejected_total",
        "wyd_handshakes_total",
        "wyd_sessions_open",
        "wyd_server_channels_published",
        "wyd_gateway_requests_total",
        "wyd_gateway_uptime_seconds",
    ] {
        assert!(
            exposto.contains(obrigatoria),
            "exposição não publica {obrigatoria}; o dashboard/regra que a cita mostraria vazio"
        );
    }

    let mut citadas = Vec::new();
    let mut dashboards = 0;
    let dir = raiz.join("monitoring/grafana/dashboards");
    for entrada in std::fs::read_dir(&dir).expect("diretório de dashboards") {
        let caminho = entrada.expect("entrada").path();
        if caminho.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        dashboards += 1;
        let bruto = std::fs::read_to_string(&caminho).expect("dashboard legível");
        let json: Value = serde_json::from_str(&bruto).expect("dashboard JSON válido");
        let loki = json["panels"].as_array().is_some_and(|paineis| {
            paineis
                .iter()
                .any(|painel| painel["datasource"]["uid"] == "loki")
        });
        if loki {
            let variaveis: Vec<&str> = json["templating"]["list"]
                .as_array()
                .map(|lista| lista.iter().filter_map(|v| v["name"].as_str()).collect())
                .unwrap_or_default();
            assert!(
                variaveis.contains(&"service"),
                "{}: dashboard Loki sem a variável `service`; os exprs usam $service",
                caminho.display()
            );
        }
        for painel in json["panels"].as_array().expect("panels") {
            let titulo = painel["title"].as_str().unwrap_or("?");
            let uid = painel["datasource"]["uid"].as_str().unwrap_or("prometheus");
            for target in painel["targets"].as_array().into_iter().flatten() {
                let Some(expr) = target["expr"].as_str() else {
                    continue;
                };
                if uid == "loki" {
                    assert!(
                        expr.contains("job=\"docker\"") && expr.contains("service"),
                        "painel {titulo}: expr Loki sem job=docker/service: {expr}"
                    );
                } else {
                    for token in tokens_de_metrica(expr) {
                        citadas.push((
                            caminho.clone(),
                            titulo.to_string(),
                            expr.to_string(),
                            token,
                        ));
                    }
                }
            }
        }
    }
    assert!(
        dashboards > 0,
        "nenhum dashboard encontrado em {}",
        dir.display()
    );

    let alertas = ler(&raiz, "monitoring/prometheus/alerts.yml");
    for linha in alertas.lines() {
        let Some(expr) = linha.trim().strip_prefix("expr:") else {
            continue;
        };
        for token in tokens_de_metrica(expr) {
            citadas.push((
                PathBuf::from("monitoring/prometheus/alerts.yml"),
                "regra".to_string(),
                expr.trim().to_string(),
                token,
            ));
        }
    }
    assert!(!citadas.is_empty(), "nenhum expr Prometheus encontrado");

    for (arquivo, titulo, expr, token) in citadas {
        // `node_*` é do node-exporter (job `node`), fonte externa declarada no scrape.
        if token.starts_with("node_") {
            continue;
        }
        assert!(
            exposto.contains(&token),
            "{} · {titulo}: `{token}` não é emitido pela exposição\n  expr: {expr}",
            arquivo.display()
        );
    }
}

#[test]
fn jobs_e_alvos_dos_exprs_existem_no_scrape_config() {
    let raiz = repo_root();
    let scrape = ler(&raiz, "monitoring/prometheus/prometheus.yml");
    let jobs: HashSet<String> = scrape
        .lines()
        .filter_map(|linha| linha.trim().strip_prefix("- job_name:"))
        .map(|nome| nome.trim().to_string())
        .collect();
    assert!(
        jobs.contains("wyd-server") && jobs.contains("wyd-gateway"),
        "jobs base ausentes"
    );

    let mut exprs = Vec::new();
    let dir = raiz.join("monitoring/grafana/dashboards");
    for entrada in std::fs::read_dir(&dir).expect("dashboards") {
        let caminho = entrada.expect("entrada").path();
        if caminho.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        let json: Value =
            serde_json::from_str(&std::fs::read_to_string(&caminho).expect("leitura"))
                .expect("JSON");
        for painel in json["panels"].as_array().expect("panels") {
            if painel["datasource"]["uid"] == "loki" {
                continue;
            }
            for target in painel["targets"].as_array().into_iter().flatten() {
                if let Some(expr) = target["expr"].as_str() {
                    exprs.push(expr.to_string());
                }
            }
        }
    }
    exprs.extend(
        ler(&raiz, "monitoring/prometheus/alerts.yml")
            .lines()
            .filter_map(|linha| linha.trim().strip_prefix("expr:").map(str::to_string)),
    );

    for expr in exprs {
        for job in jobs_citados(&expr) {
            assert!(
                jobs.contains(&job),
                "expr cita job=\"{job}\" que não existe em prometheus.yml\n  expr: {expr}"
            );
        }
    }

    let alvos: Value =
        serde_json::from_str(&ler(&raiz, "monitoring/prometheus/targets/wyd-server.json"))
            .expect("targets JSON");
    let servicos = servicos_do_compose(&raiz);
    let mut canais = Vec::new();
    for alvo in alvos.as_array().expect("lista de alvos") {
        let labels = &alvo["labels"];
        let canal = labels["canal"].as_str().expect("label canal no alvo");
        canais.push(canal.to_string());
        for target in alvo["targets"].as_array().into_iter().flatten() {
            let host = target
                .as_str()
                .and_then(|t| t.split(':').next())
                .unwrap_or("");
            assert!(
                servicos.contains(host),
                "alvo `{host}` do job wyd-server não é serviço dos composes (canais: {canais:?})"
            );
        }
    }
    assert!(
        canais.len() >= 2,
        "o job wyd-server precisa de um alvo por canal; achei {canais:?}"
    );
}

#[test]
fn queries_do_portal_existem_na_exposicao_e_no_scrape() {
    let raiz = repo_root();
    let exposto = nomes_da_exposicao(&exposicao());
    let scrape = ler(&raiz, "monitoring/prometheus/prometheus.yml");
    let jobs: HashSet<String> = scrape
        .lines()
        .filter_map(|linha| linha.trim().strip_prefix("- job_name:"))
        .map(|nome| nome.trim().to_string())
        .collect();

    let queries = queries_do_portal(&ler(&raiz, "wyd-portal/src/server/monitoring.ts"));
    assert!(
        queries.len() >= 12,
        "extraí só {} queries de monitoring.ts; o parser precisa acompanhar o arquivo",
        queries.len()
    );
    for expr in queries {
        for token in tokens_de_metrica(&expr) {
            if token.starts_with("node_") {
                continue;
            }
            assert!(
                exposto.contains(&token),
                "wyd-portal/src/server/monitoring.ts: `{token}` não é emitido pela exposição\n  expr: {expr}"
            );
        }
        for job in jobs_citados(&expr) {
            assert!(
                jobs.contains(&job),
                "wyd-portal/src/server/monitoring.ts: job=\"{job}\" não existe em prometheus.yml\n  expr: {expr}"
            );
        }
    }
}

#[test]
fn metrics_servido_por_http_traz_o_contrato_do_motor() {
    let motor = Arc::new(Metrics::new());
    motor.record_tick(1_500, 3);
    let servico = Arc::new(ServiceMetrics::new(2));
    servico.attach_engine(Arc::clone(&motor));

    let listener = std::net::TcpListener::bind("127.0.0.1:0").expect("porta livre");
    let endereco = listener.local_addr().expect("endereço");
    std::thread::spawn(move || wyd_server::metrics::serve(listener, servico));

    let mut stream = TcpStream::connect(endereco).expect("conecta no /metrics");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream
        .write_all(b"GET /metrics HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
        .expect("GET");
    let mut resposta = String::new();
    stream.read_to_string(&mut resposta).expect("corpo");

    assert!(
        resposta.starts_with("HTTP/1.1 200"),
        "resposta inesperada: {}",
        &resposta[..resposta.len().min(200)]
    );
    for nome in [
        "wyd_ticks_total",
        "wyd_tick_duration_microseconds_bucket",
        "wyd_tick_budget_microseconds",
        "wyd_entities_active",
        "process_resident_memory_bytes",
        "wyd_packets_total",
    ] {
        assert!(resposta.contains(nome), "/metrics HTTP não trouxe {nome}");
    }
}
