//! Pipeline de lançamento compartilhado entre a CLI e a janela Tauri (CLIENT-08).
//!
//! Fail-closed (ADR-0003): manifesto vermelho não escreve nada e não abre o jogo.
//! Toda a escrita no disco continua em `manifest`/`serverlist`/`snbin`; aqui só
//! vive a ordem das etapas e o plano de `bootstrap` — nenhuma regra duplicada.

use std::fmt;
use std::path::Path;

use crate::{bootstrap, identidade, manifest, serverlist, snbin};

/// Manifesto procurado na pasta do jogo quando o chamador não informa outro.
pub const MANIFESTO_PADRAO: &str = "manifest.txt";
/// Registro de canais procurado na pasta do jogo.
pub const REGISTRO_PADRAO: &str = "canais.txt";

/// Recusas do pipeline de lançamento.
#[derive(Debug)]
pub enum LancamentoError {
    /// Manifesto ilegível ou fora do formato.
    Manifesto(manifest::ManifestError),
    /// Um ou mais arquivos do manifesto não conferem (lista `caminho: status`).
    Verificacao(Vec<String>),
    /// Registro inválido ou falha ao instalar o `serverlist.bin`.
    Serverlist(serverlist::ServerlistError),
    /// Identidade recusada ao montar o `sn.bin`.
    Snbin(String),
    /// Executável ausente ou falha ao iniciar o processo.
    Bootstrap(bootstrap::LaunchError),
}

impl fmt::Display for LancamentoError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Manifesto(e) => write!(f, "manifesto inacessível/inválido — nada escrito: {e}"),
            Self::Verificacao(falhas) => {
                write!(f, "verificação falhou — jogo não será iniciado:")?;
                for falha in falhas {
                    write!(f, "\n  {falha}")?;
                }
                Ok(())
            }
            Self::Serverlist(e) => write!(f, "serverlist recusado — jogo não será iniciado: {e}"),
            Self::Snbin(e) => write!(f, "sn.bin recusado — jogo não será iniciado: {e}"),
            Self::Bootstrap(e) => write!(f, "lançamento impossível: {e}"),
        }
    }
}

impl std::error::Error for LancamentoError {}

/// Tudo que o lançamento precisa, já calculado e verificado (nada escrito ainda).
#[derive(Debug)]
pub struct Plano {
    /// Build declarado no manifesto.
    pub build: u32,
    /// Células do registro de canais.
    pub cells: Vec<serverlist::Cell>,
    /// Bytes do `serverlist.bin` a instalar.
    pub serverlist: Vec<u8>,
    /// Bytes do `sn.bin` da identidade.
    pub sn: Vec<u8>,
    /// Executável e argumentos do jogo.
    pub launch: bootstrap::Launch,
}

/// Resultado da escrita no diretório do jogo.
#[derive(Debug)]
pub struct Preparo {
    /// Relatório do `serverlist.bin`.
    pub instalacao: serverlist::InstallReport,
    /// Relatório do `sn.bin`.
    pub sn: crate::branding::InstallReport,
}

/// Resultado completo de abrir o jogo.
#[derive(Debug)]
pub struct Relatorio {
    pub build: u32,
    pub cells: Vec<serverlist::Cell>,
    pub instalacao: serverlist::InstallReport,
    pub sn: crate::branding::InstallReport,
    pub pids: Vec<u32>,
}

/// Verifica o manifesto, monta `serverlist`/`sn.bin` e planeja o executável.
///
/// Não escreve nada e não executa nada: é o caminho do `--dry-run` e a etapa de
/// recusa do fail-closed.
pub fn planejar(
    dir: &Path,
    manifesto: &Path,
    registro: &Path,
    identidade: &identidade::Identidade,
    exe: &str,
    extras: &[String],
) -> Result<Plano, LancamentoError> {
    let m = manifest::load(manifesto).map_err(LancamentoError::Manifesto)?;
    let rel = manifest::verify(&m, dir);
    if !rel.clean() {
        let falhas = rel
            .failures()
            .map(|f| format!("{}: {}", f.path, f.status))
            .collect();
        return Err(LancamentoError::Verificacao(falhas));
    }
    let serverlist =
        serverlist::generate_from_registry(registro).map_err(LancamentoError::Serverlist)?;
    let cells = serverlist::decode(&serverlist).map_err(LancamentoError::Serverlist)?;
    let sn = snbin::build(identidade, &snbin::canais_por_grupo(&cells))
        .map_err(|e| LancamentoError::Snbin(e.to_string()))?;
    let launch = bootstrap::plan_path(dir, exe, extras).map_err(LancamentoError::Bootstrap)?;
    Ok(Plano {
        build: m.build,
        cells,
        serverlist,
        sn,
        launch,
    })
}

/// Instala `serverlist.bin` e `sn.bin` de forma atômica com backup.
pub fn preparar(dir: &Path, plano: &Plano) -> Result<Preparo, LancamentoError> {
    let instalacao =
        serverlist::install(dir, &plano.serverlist).map_err(LancamentoError::Serverlist)?;
    let sn = snbin::install(dir, &plano.sn).map_err(|e| LancamentoError::Snbin(e.to_string()))?;
    Ok(Preparo { instalacao, sn })
}

/// Abre `instancias` janelas do plano já verificado.
///
/// Multiboxing é prática normal do WYD: janelas independentes, sem trava única.
pub fn abrir(plano: &Plano, instancias: usize) -> Result<Vec<u32>, LancamentoError> {
    let mut pids = Vec::new();
    for _ in 0..instancias {
        let filho = bootstrap::spawn(&plano.launch).map_err(LancamentoError::Bootstrap)?;
        pids.push(filho.id());
    }
    Ok(pids)
}

/// Caminho completo usado pela janela: planejar → preparar → abrir.
pub fn iniciar(
    dir: &Path,
    manifesto: &Path,
    registro: &Path,
    identidade: &identidade::Identidade,
    exe: &str,
    extras: &[String],
    instancias: usize,
) -> Result<Relatorio, LancamentoError> {
    let plano = planejar(dir, manifesto, registro, identidade, exe, extras)?;
    let preparo = preparar(dir, &plano)?;
    let pids = abrir(&plano, instancias)?;
    Ok(Relatorio {
        build: plano.build,
        cells: plano.cells,
        instalacao: preparo.instalacao,
        sn: preparo.sn,
        pids,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sha256;
    use std::fs;
    use std::path::PathBuf;

    const REGISTRO: &str =
        "# canal de teste\n1 0 http://192.168.100.3/serv00.htm\n1 1 192.168.100.3\n";

    fn tmpdir(tag: &str) -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static N: AtomicU64 = AtomicU64::new(0);
        let dir = std::env::temp_dir().join(format!(
            "wyd-launcher-lancamento-{tag}-{}-{}",
            std::process::id(),
            N.fetch_add(1, Ordering::Relaxed)
        ));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).expect("tmp");
        dir
    }

    fn escrever_manifesto(dir: &Path, conteudo: &[u8]) -> PathBuf {
        fs::write(dir.join("jogo.txt"), conteudo).expect("jogo");
        let texto = format!(
            "build 7\nprotocol 7662\n{} {} jogo.txt\n",
            sha256::hex_of(conteudo),
            conteudo.len()
        );
        let path = dir.join(MANIFESTO_PADRAO);
        fs::write(&path, texto).expect("manifesto");
        path
    }

    fn registro(dir: &Path) -> PathBuf {
        let path = dir.join(REGISTRO_PADRAO);
        fs::write(&path, REGISTRO).expect("registro");
        path
    }

    #[test]
    fn planejar_verifica_manifesto_e_monta_plano() {
        let dir = tmpdir("ok");
        let manifesto = escrever_manifesto(&dir, b"cliente v1");
        let registro = registro(&dir);
        fs::write(dir.join("WYD.exe"), b"exe falso").expect("exe");
        let id = identidade::padrao().expect("identidade");
        let plano = planejar(&dir, &manifesto, &registro, &id, "WYD.exe", &[]).expect("plano");
        assert_eq!(plano.build, 7);
        assert_eq!(plano.cells.len(), 2);
        assert_eq!(plano.launch.exe, dir.join("WYD.exe"));
        assert_eq!(plano.serverlist.len(), serverlist::BIN_SIZE);
        assert!(!plano.sn.is_empty());
        assert!(
            !dir.join(serverlist::FILE_NAME).exists(),
            "planejar não escreve no diretório do jogo"
        );
    }

    #[test]
    fn manifesto_vermelho_e_recusa_sem_escrever() {
        let dir = tmpdir("vermelho");
        let manifesto = escrever_manifesto(&dir, b"cliente v1");
        fs::write(dir.join("jogo.txt"), b"adulterado").expect("adultera");
        fs::write(dir.join("WYD.exe"), b"exe falso").expect("exe");
        let id = identidade::padrao().expect("identidade");
        let erro =
            planejar(&dir, &manifesto, &registro(&dir), &id, "WYD.exe", &[]).expect_err("recusa");
        match erro {
            LancamentoError::Verificacao(falhas) => {
                assert_eq!(falhas.len(), 1);
                assert!(falhas[0].starts_with("jogo.txt: "), "{falhas:?}");
            }
            outro => panic!("erro inesperado: {outro}"),
        }
        assert!(!dir.join(serverlist::FILE_NAME).exists());
        assert!(!dir.join(snbin::FILE_NAME).exists());
    }

    #[test]
    fn manifesto_ausente_e_erro_tipado() {
        let dir = tmpdir("sem-manifesto");
        let id = identidade::padrao().expect("identidade");
        let erro = planejar(
            &dir,
            &dir.join("nao-existe.txt"),
            &registro(&dir),
            &id,
            "WYD.exe",
            &[],
        )
        .expect_err("recusa");
        assert!(matches!(erro, LancamentoError::Manifesto(_)));
    }

    #[test]
    fn exe_ausente_impede_o_plano() {
        let dir = tmpdir("sem-exe");
        let manifesto = escrever_manifesto(&dir, b"cliente v1");
        let id = identidade::padrao().expect("identidade");
        let erro =
            planejar(&dir, &manifesto, &registro(&dir), &id, "WYD.exe", &[]).expect_err("recusa");
        assert!(matches!(erro, LancamentoError::Bootstrap(_)));
    }

    #[test]
    fn preparar_instala_serverlist_e_sn_uma_vez() {
        let dir = tmpdir("preparar");
        let manifesto = escrever_manifesto(&dir, b"cliente v1");
        let id = identidade::padrao().expect("identidade");
        fs::write(dir.join("WYD.exe"), b"exe falso").expect("exe");
        let plano =
            planejar(&dir, &manifesto, &registro(&dir), &id, "WYD.exe", &[]).expect("plano");
        let primeiro = preparar(&dir, &plano).expect("prepara");
        assert!(primeiro.instalacao.changed);
        assert!(primeiro.sn.changed);
        assert!(dir.join(serverlist::FILE_NAME).exists());
        assert!(dir.join(snbin::FILE_NAME).exists());
        let segundo = preparar(&dir, &plano).expect("reprepara");
        assert!(
            !segundo.instalacao.changed,
            "nada reescrito quando idêntico"
        );
        assert!(!segundo.sn.changed, "nada reescrito quando idêntico");
    }
}
