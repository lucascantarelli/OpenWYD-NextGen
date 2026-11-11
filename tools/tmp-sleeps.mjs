import fs from 'node:fs';

const arquivos = {
  'crates/wyd-server/tests/sistema/selecao.rs': `    let inicio = std::time::Instant::now();
    let mut ultimo = None;
    while inicio.elapsed() < limite {
        if let Some(valor) = ler_gauge_entidades(prometheus, canal) {
            ultimo = Some(valor);
            if aceita(valor) {
                return Some(valor);
            }
        }
        std::thread::sleep(Duration::from_secs(2));
    }
    ultimo`,
};

const trocas = [
  [
    'crates/wyd-server/tests/sistema/selecao.rs',
    arquivos['crates/wyd-server/tests/sistema/selecao.rs'],
    `    let mut ultimo = None;
    let _ = common::esperar_ate_a_cada(limite, Duration::from_secs(1), || {
        if let Some(valor) = ler_gauge_entidades(prometheus, canal) {
            ultimo = Some(valor);
            return aceita(valor);
        }
        false
    });
    ultimo`,
  ],
  [
    'crates/wyd-server/tests/sistema/sessao_unica.rs',
    `    let limite = Instant::now() + Duration::from_secs(10);
    let mut entrou = false;
    while Instant::now() < limite && !entrou {
        std::thread::sleep(Duration::from_millis(400));
        let (_sessao3, ok) = entra(&canal2, &conta);
        entrou = ok;
        if entrou {
            break;
        }
    }`,
    `    let entrou = common::esperar_ate(Duration::from_secs(10), || {
        let (_sessao3, ok) = entra(&canal2, &conta);
        ok
    });`,
  ],
  [
    'crates/wyd-server/tests/sistema/limite_ip.rs',
    `    // Dá tempo do accept registrar a vaga.
    std::thread::sleep(Duration::from_millis(200));
    let mut segunda = conecta(porta);
    assert!(
        recusada(&mut segunda),
        "segunda conexão do mesmo IP tem de ser fechada sem resposta"
    );`,
    `    // A vaga é registrada pelo accept: tenta até a segunda conexão ser recusada.
    let mut segunda = conecta(porta);
    let recusou = common::esperar_ate(Duration::from_secs(2), || recusada(&mut segunda));
    assert!(
        recusou,
        "segunda conexão do mesmo IP tem de ser fechada sem resposta"
    );`,
  ],
  [
    'crates/wyd-server/tests/sistema/limite_ip.rs',
    `    let limite = std::time::Instant::now() + Duration::from_secs(5);
    let mut terceira = loop {
        let mut tentativa = conecta(porta);
        if !recusada(&mut tentativa) {
            break tentativa;
        }
        assert!(
            std::time::Instant::now() < limite,
            "vaga não voltou após a primeira conexão sair"
        );
        std::thread::sleep(Duration::from_millis(50));
    };
    let _ = terceira.write_all(&INITCODE.to_le_bytes());`,
    `    let mut terceira = None;
    let liberou = common::esperar_ate(Duration::from_secs(5), || {
        let mut tentativa = conecta(porta);
        if recusada(&mut tentativa) {
            false
        } else {
            terceira = Some(tentativa);
            true
        }
    });
    assert!(liberou, "vaga não voltou após a primeira conexão sair");
    let mut terceira = terceira.expect("terceira conexão");
    let _ = terceira.write_all(&INITCODE.to_le_bytes());`,
  ],
  [
    'crates/wyd-server/tests/sistema/selchar.rs',
    `    let inicio = std::time::Instant::now();
    loop {
        let viva = rt
            .block_on(backend.live_session_of(&conta))
            .expect("sessão da conta");
        if viva.is_none() {
            break;
        }
        assert!(
            inicio.elapsed() < Duration::from_secs(5),
            "sessão continuou viva depois do logout explícito ({viva:?})"
        );
        std::thread::sleep(Duration::from_millis(100));
    }`,
    `    let saiu = common::esperar_ate(Duration::from_secs(5), || {
        rt.block_on(backend.live_session_of(&conta))
            .expect("sessão da conta")
            .is_none()
    });
    assert!(saiu, "sessão continuou viva depois do logout explícito");`,
  ],
  [
    'crates/wyd-server/tests/integracao/tick_processo.rs',
    `    let (mut thread, _handle) = EngineThread::spawn(engine, 10).expect("thread do tick");
    std::thread::sleep(Duration::from_millis(300));
    thread.stop_and_join();`,
    `    let (mut thread, _handle) = EngineThread::spawn(engine, 10).expect("thread do tick");
    assert!(
        common::esperar_ate(Duration::from_secs(2), || metrics.ticks() > 0),
        "o processo não executou tick dentro do prazo"
    );
    thread.stop_and_join();`,
  ],
  [
    'crates/wyd-server/tests/integracao/tick_processo.rs',
    `    // Parada limpa: nenhum tick novo depois do join.
    std::thread::sleep(Duration::from_millis(120));
    assert_eq!(metrics.ticks(), ticks, "sem tick órfão após a parada");`,
    `    // Parada limpa: durante 150 ms nenhum tick novo pode aparecer.
    assert!(
        common::esperar_ate(Duration::from_millis(150), || metrics.ticks() == ticks),
        "sem tick órfão após a parada"
    );`,
  ],
  [
    'crates/wyd-server/tests/integracao/config_recarga.rs',
    `    let alvo = source.revision();
    let limite = Instant::now() + Duration::from_secs(5);
    while live.revision() < alvo && Instant::now() < limite {
        std::thread::sleep(Duration::from_millis(10));
    }`,
    `    let alvo = source.revision();
    assert!(
        common::esperar_ate(Duration::from_secs(5), || live.revision() >= alvo),
        "recarga não publicou a revisão {alvo}"
    );`,
  ],
  [
    'crates/wyd-server/tests/integracao/config_recarga.rs',
    `    let join = Instant::now() + Duration::from_secs(5);
    while !thread.is_finished() && Instant::now() < join {
        std::thread::sleep(Duration::from_millis(5));
    }
    assert!(thread.is_finished(), "thread de recarga não parou");`,
    `    assert!(
        common::esperar_ate(Duration::from_secs(5), || thread.is_finished()),
        "thread de recarga não parou"
    );`,
  ],
];

for (const [arquivo, antigo, novo] of trocas) {
  const texto = fs.readFileSync(arquivo, 'utf8');
  if (!texto.includes(antigo)) {
    throw new Error(`trecho não encontrado em ${arquivo}`);
  }
  fs.writeFileSync(arquivo, texto.replace(antigo, novo));
}

// Módulo comum nos arquivos de polling que ainda não o importam.
for (const arquivo of [
  'crates/wyd-server/tests/sistema/selecao.rs',
  'crates/wyd-server/tests/sistema/sessao_unica.rs',
  'crates/wyd-server/tests/sistema/limite_ip.rs',
  'crates/wyd-server/tests/integracao/tick_processo.rs',
  'crates/wyd-server/tests/integracao/config_recarga.rs',
]) {
  let texto = fs.readFileSync(arquivo, 'utf8');
  if (texto.includes('mod common;')) continue;
  const linhas = texto.split('\n');
  let fim = 0;
  while (fim < linhas.length && linhas[fim].startsWith('//!')) fim += 1;
  linhas.splice(fim, 0, '', '#[path = "../common/mod.rs"]', 'mod common;');
  fs.writeFileSync(arquivo, linhas.join('\n'));
}

console.log('trocas', trocas.length);
