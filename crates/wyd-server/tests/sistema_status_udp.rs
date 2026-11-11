//! Sonda UDP de status do canal: o cliente 7662 manda **4 bytes** para `host:8281` e só
//! marca o canal como acessível quando o mesmo datagrama volta (`target/teste-udp.json`).
//!
//! O listener real é `wyd_server::status::serve_udp`; o teste usa porta efêmera e prova os
//! dois lados: eco do probe e silêncio para datagrama de tamanho inesperado.

use std::io::ErrorKind;
use std::net::{SocketAddr, UdpSocket};
use std::time::Duration;

use wyd_server::status::{serve_udp, UDP_PROBE_LEN};

/// Sobe o listener em porta efêmera e devolve o cliente pronto para sondar.
fn sonda() -> (UdpSocket, SocketAddr) {
    let listener = UdpSocket::bind("127.0.0.1:0").expect("porta efêmera do canal");
    let endereco = listener.local_addr().expect("endereço do canal");
    std::thread::spawn(move || serve_udp(listener));
    let cliente = UdpSocket::bind("127.0.0.1:0").expect("porta efêmera do cliente");
    cliente
        .set_read_timeout(Some(Duration::from_secs(2)))
        .expect("timeout de leitura");
    (cliente, endereco)
}

#[test]
fn probe_de_status_volta_igual_ao_enviado() {
    let (cliente, endereco) = sonda();
    let probe = [0x01u8, 0x02, 0x03, 0x04];
    cliente.send_to(&probe, endereco).expect("envia a sonda");

    let mut resposta = [0u8; 64];
    let (lido, de) = cliente.recv_from(&mut resposta).expect("resposta da sonda");
    assert_eq!(lido, UDP_PROBE_LEN, "o canal devolve os 4 bytes da sonda");
    assert_eq!(&resposta[..lido], &probe, "eco byte a byte da sonda");
    assert_eq!(de, endereco, "resposta veio do canal sondado");
}

#[test]
fn datagrama_de_outro_tamanho_nao_recebe_resposta() {
    let (cliente, endereco) = sonda();
    cliente
        .send_to(&[0x01u8, 0x02], endereco)
        .expect("envia datagrama inválido");

    let mut resposta = [0u8; 64];
    match cliente.recv_from(&mut resposta) {
        Err(error) => assert!(
            matches!(error.kind(), ErrorKind::WouldBlock | ErrorKind::TimedOut),
            "esperado silêncio, veio {error:?}"
        ),
        Ok((lido, _)) => panic!("canal respondeu {lido} B a datagrama inválido"),
    }
}
