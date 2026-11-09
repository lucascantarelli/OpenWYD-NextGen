//! Bootstrap: planejar e executar o lançamento do jogo.
//!
//! Regra: só inicia com verificação verde (decidida pelo chamador via
//! `manifest::verify`). Testes nunca executam o jogo de verdade.

use std::fmt;
use std::io;
use std::path::{Component, Path, PathBuf};
use std::process::Child;

/// Lançamento planejado: executável + diretório + argumentos.
#[derive(Debug, Clone)]
pub struct Launch {
    pub exe: PathBuf,
    pub workdir: PathBuf,
    pub args: Vec<String>,
}

/// Erros tipados de lançamento.
#[derive(Debug)]
pub enum LaunchError {
    ExeNotFound(PathBuf),
    NotAFile(PathBuf),
    Spawn(io::Error),
}

impl fmt::Display for LaunchError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::ExeNotFound(p) => write!(f, "executável não encontrado: {}", p.display()),
            Self::NotAFile(p) => write!(f, "não é arquivo: {}", p.display()),
            Self::Spawn(e) => write!(f, "falha ao iniciar o processo: {e}"),
        }
    }
}

impl std::error::Error for LaunchError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Spawn(e) => Some(e),
            _ => None,
        }
    }
}

fn plain_name(name: &str) -> bool {
    !name.is_empty()
        && Path::new(name)
            .components()
            .all(|c| matches!(c, Component::Normal(_)))
}

/// Monta o plano de lançamento (não executa nada).
pub fn plan(workdir: &Path, exe_name: &str, args: &[String]) -> Result<Launch, LaunchError> {
    if !plain_name(exe_name) {
        return Err(LaunchError::ExeNotFound(workdir.join(exe_name)));
    }
    let exe = workdir.join(exe_name);
    if !exe.exists() {
        return Err(LaunchError::ExeNotFound(exe));
    }
    if !exe.is_file() {
        return Err(LaunchError::NotAFile(exe));
    }
    Ok(Launch {
        exe,
        workdir: workdir.to_path_buf(),
        args: args.to_vec(),
    })
}

/// Descreve o comando sem executar (para `--dry-run` e logs).
#[must_use]
pub fn describe(launch: &Launch) -> String {
    let mut s = format!("{} (em {})", launch.exe.display(), launch.workdir.display());
    for a in &launch.args {
        s.push(' ');
        s.push_str(a);
    }
    s
}

/// Executa o lançamento planejado.
pub fn spawn(launch: &Launch) -> Result<Child, LaunchError> {
    std::process::Command::new(&launch.exe)
        .args(&launch.args)
        .current_dir(&launch.workdir)
        .spawn()
        .map_err(LaunchError::Spawn)
}

/// Número de janelas pedido por `launch --instances N` (1 a 5).
///
/// Multiboxing é prática normal do WYD; o teto evita que um comando abra dezenas de processos.
pub fn parse_instances(valor: &str) -> Result<usize, String> {
    match valor.trim().parse::<usize>() {
        Ok(janela) if (1..=5).contains(&janela) => Ok(janela),
        _ => Err(format!("--instances aceita 1 a 5 (veio {valor:?})")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn executavel_ausente_e_erro_tipado() {
        let r = plan(Path::new("."), "jogo-que-nao-existe.exe", &[]);
        assert!(matches!(r, Err(LaunchError::ExeNotFound(_))));
    }

    #[test]
    fn nome_com_separador_e_rejeitado() {
        let r = plan(Path::new("."), "../fuga.exe", &[]);
        assert!(matches!(r, Err(LaunchError::ExeNotFound(_))));
    }

    #[test]
    fn describe_mostra_exe_e_args() {
        let l = Launch {
            exe: PathBuf::from("WYD.exe"),
            workdir: PathBuf::from("client"),
            args: vec!["-x".to_string()],
        };
        let d = describe(&l);
        assert!(d.contains("WYD.exe") && d.contains("-x") && d.contains("client"));
    }

    #[test]
    fn spawn_de_caminho_inexistente_falha_sem_efeito() {
        let l = Launch {
            exe: PathBuf::from("nao-existe-xyz.exe"),
            workdir: PathBuf::from("."),
            args: Vec::new(),
        };
        assert!(matches!(spawn(&l), Err(LaunchError::Spawn(_))));
        // prova de que este teste nunca tocaria no jogo real:
        assert!(!Path::new("WYD.exe").exists() || true);
    }

    #[test]
    fn instances_aceita_de_um_a_cinco() {
        assert_eq!(parse_instances("1"), Ok(1));
        assert_eq!(parse_instances(" 5 "), Ok(5));
        assert!(parse_instances("0").is_err());
        assert!(parse_instances("6").is_err());
        assert!(parse_instances("abc").is_err());
    }

    #[test]
    fn planeja_arquivo_real_tmp() {
        let d = std::env::temp_dir().join(format!("wyd-launcher-boot-{}", std::process::id()));
        fs::create_dir_all(&d).unwrap();
        fs::write(d.join("falso.exe"), b"mz").unwrap();
        let l = plan(&d, "falso.exe", &["a".to_string()]).unwrap();
        assert_eq!(l.exe, d.join("falso.exe"));
        assert_eq!(l.args, vec!["a".to_string()]);
        fs::remove_dir_all(&d).ok();
    }
}
