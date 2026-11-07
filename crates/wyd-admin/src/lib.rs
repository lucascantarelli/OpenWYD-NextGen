//! Control plane Cold Path (esqueleto).
//!
//! Fala com o portal Next.js de forma assincrona, isolada do tick loop.

/// Prefixo das rotas administrativas neste esqueleto.
pub const ADMIN_SCOPE: &str = "/admin";

/// Verifica se o caminho pertence ao escopo admin.
#[must_use]
pub fn is_admin_path(path: &str) -> bool {
    path == ADMIN_SCOPE || path.starts_with("/admin/")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_admin_scope_only() {
        assert!(is_admin_path("/admin"));
        assert!(is_admin_path("/admin/users"));
        assert!(!is_admin_path("/player"));
    }
}
