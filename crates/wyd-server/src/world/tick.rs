//! Submódulo de `world` (ADR-0026, etapa 3).

use super::*;

impl Engine {
    /// Aplica um passo de movimento autoritativo (`_MSG_Action`, `Route[24]` do legado).
    ///
    /// O cliente manda **destino**; orçamento por tempo, retângulo do mapa e teto de 24 tiles
    /// por pacote são do servidor (`MoveRules`). Aceitar move a entidade no SoA e grava a nova
    /// posição no estado vivo, que é o que vai para o save.
    pub fn move_to(
        &mut self,
        entity: Entity,
        to: Position,
        now_ms: u64,
    ) -> Result<Position, CoreError> {
        let Some(atual) = self.store.position_of(entity) else {
            return Err(CoreError::EntityNotAlive {
                index: entity.index,
                generation: entity.generation,
            });
        };
        if !self.move_rules.contains(to) {
            return Err(CoreError::MoveTooFar {
                distance: movement_system::distance(atual, to),
                allowed: MOVE_MAX_TILES,
            });
        }
        // Primeiro passo da entidade usa 1 s de orçamento; depois, o tempo real desde o
        // último passo aceito (cap de 1 s: o que ficou parado não vira teleporte).
        let anterior = self
            .last_move
            .get(&entity.index)
            .copied()
            .unwrap_or_else(|| now_ms.saturating_sub(1_000));
        let elapsed_ms = now_ms.saturating_sub(anterior).min(1_000);
        // Um clique de 20 tiles é destino, não teleporte: anda o que o orçamento permite
        // agora e o resto nos ticks (`walk_pending`). Recusar o clique inteiro era o bug que
        // travava o personagem no spawn e deixava toda loja de Armia fora de alcance.
        let budget = u32::try_from(self.move_rules.budget(elapsed_ms))
            .unwrap_or(u32::MAX)
            .max(1);
        let alvo = step_toward(atual, to, budget);
        let posicao =
            movement_system::step(&mut self.store, entity, alvo, elapsed_ms, self.move_rules)?;
        if let Some(state) = self.live.get_mut(&entity.index) {
            state.x = posicao.x;
            state.y = posicao.y;
        }
        self.last_move.insert(entity.index, now_ms);
        if posicao == to {
            self.destinations.remove(&entity.index);
        } else {
            self.destinations.insert(entity.index, (entity, to));
        }
        Ok(posicao)
    }

    /// Anda um orçamento por tick até o destino pendente de cada entidade.
    ///
    /// O passo usa o mesmo `movement_system::step` (orçamento por tempo + retângulo do
    /// mapa): destino fora do mapa ou entidade morta cancela a caminhada, sem teleporte.
    fn walk_pending(&mut self, elapsed_ms: u64) {
        if self.destinations.is_empty() {
            return;
        }
        let pendentes: Vec<(u32, Entity, Position)> = self
            .destinations
            .iter()
            .map(|(indice, (entity, destino))| (*indice, *entity, *destino))
            .collect();
        let elapsed_ms = elapsed_ms.min(1_000);
        let budget = u32::try_from(self.move_rules.budget(elapsed_ms))
            .unwrap_or(u32::MAX)
            .max(1);
        for (indice, entity, destino) in pendentes {
            let Some(atual) = self.store.position_of(entity) else {
                self.destinations.remove(&indice);
                continue;
            };
            if atual == destino {
                self.destinations.remove(&indice);
                continue;
            }
            let alvo = step_toward(atual, destino, budget);
            match movement_system::step(&mut self.store, entity, alvo, elapsed_ms, self.move_rules)
            {
                Ok(posicao) => {
                    if let Some(state) = self.live.get_mut(&entity.index) {
                        state.x = posicao.x;
                        state.y = posicao.y;
                    }
                    if posicao == destino {
                        self.destinations.remove(&indice);
                    }
                }
                Err(_) => {
                    self.destinations.remove(&indice);
                }
            }
        }
    }

    /// Aplica um comando da borda sem bloquear o tick.
    fn apply(&mut self, command: Command) {
        match command {
            Command::Enter {
                character,
                skin,
                reply,
            } => {
                let _ = reply.send(self.enter(&character, *skin));
            }
            Command::Leave { entity, reply } => {
                let _ = reply.send(self.leave(entity));
            }
            Command::Cp { entity, reply } => {
                let _ = reply.send(self.cp_of(entity));
            }
            Command::Snapshots { reply } => {
                let _ = reply.send(self.snapshots());
            }
            Command::Npcs { reply } => {
                let _ = reply.send(self.scene_snapshot());
            }
            Command::SceneAround {
                speaker,
                radius,
                reply,
            } => {
                let cena = self
                    .store
                    .position_of(speaker)
                    .map(|centro| self.scene_around(centro, radius));
                let _ = reply.send(cena);
            }
            Command::SyncNpcs { specs, reply } => {
                let _ = reply.send(self.sync_npcs(specs));
            }
            Command::Attack {
                attacker,
                x,
                y,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.attack_at(attacker, x, y, now_ms));
            }
            Command::UseKey {
                speaker,
                x,
                y,
                keys,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.use_key_at(speaker, x, y, &keys, now_ms));
            }
            Command::Move {
                entity,
                x,
                y,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.move_to(entity, Position { x, y }, now_ms));
            }
            Command::Talk {
                speaker,
                x,
                y,
                reply,
            } => {
                let _ = reply.send(self.talk_at(speaker, x, y));
            }
        }
    }

    /// Um passo fixo de simulação: drena eventos parados e publica o gauge de entidades.
    fn step(&mut self) {
        self.events_last_tick = self.store.events().len();
        if self.events_last_tick > 0 {
            // Consumidor de borda entra com o fio de cena (03-9); até lá, drenar impede a
            // fila de crescer.
            self.events_last_tick = self.store.drain_events().len();
        }
    }

    /// Avança o tempo real acumulado e executa os passos devidos (0..=5).
    pub fn advance(&mut self, elapsed_ms: u64) -> u64 {
        self.walk_pending(elapsed_ms);
        let steps = self.scheduler.advance(elapsed_ms);
        for _ in 0..steps {
            let inicio = Instant::now();
            self.step();
            let micros = u64::try_from(inicio.elapsed().as_micros()).unwrap_or(u64::MAX);
            let entidades = u64::try_from(self.store.live_count()).unwrap_or(u64::MAX);
            self.metrics.record_tick(micros, entidades);
        }
        steps
    }

    /// Tick atual do processo.
    #[must_use]
    pub fn tick(&self) -> Tick {
        self.scheduler.tick()
    }

    /// Entidades vivas no mundo.
    #[must_use]
    pub fn entities(&self) -> usize {
        self.store.live_count()
    }

    /// Personagens vivos (jogadores) no mundo.
    #[must_use]
    pub fn players(&self) -> usize {
        self.live.len()
    }

    /// Eventos drenados no último passo (consumidor definitivo em 03-9).
    #[must_use]
    pub fn events_last_tick(&self) -> usize {
        self.events_last_tick
    }

    /// Leitura do armazém para os sistemas do core.
    #[must_use]
    pub fn store(&self) -> &ComponentStore {
        &self.store
    }

    /// Registro de métricas do motor (compartilhado com o `/metrics` do processo).
    #[must_use]
    pub fn metrics(&self) -> &Arc<Metrics> {
        &self.metrics
    }
}

/// Thread do tick com parada limpa — sem `unsafe`, sem `tokio` no laço.
#[derive(Debug)]
pub struct EngineThread {
    stop: Arc<AtomicBool>,
    handle: Option<JoinHandle<()>>,
    metrics: Arc<Metrics>,
}

impl EngineThread {
    /// Sobe a thread `wyd-tick` e devolve a porta da borda.
    ///
    /// `interval_ms` é o intervalo do scheduler (50 ms a 20 Hz). Criar thread é operação de
    /// boot: falha de SO vira `Err` e o processo não finge que está simulando.
    pub fn spawn(mut engine: Engine, interval_ms: u64) -> Result<(Self, EngineHandle), String> {
        let (tx, rx) = mpsc::channel::<Command>();
        let handle = EngineHandle { tx };
        let stop = Arc::new(AtomicBool::new(false));
        let flag = Arc::clone(&stop);
        let metrics = Arc::clone(engine.metrics());
        let resumo_metrics = Arc::clone(&metrics);
        let intervalo = Duration::from_millis(interval_ms.max(1));
        let thread_handle = thread::Builder::new()
            .name("wyd-tick".to_string())
            .spawn(move || {
                run_loop(&mut engine, &rx, &flag, intervalo, &resumo_metrics);
            })
            .map_err(|error| format!("thread do tick: {error}"))?;
        Ok((
            Self {
                stop,
                handle: Some(thread_handle),
                metrics,
            },
            handle,
        ))
    }

    /// Registro do motor (para o `/metrics` do processo).
    #[must_use]
    pub fn metrics(&self) -> &Arc<Metrics> {
        &self.metrics
    }

    /// Sinaliza parada e espera a thread terminar (idempotente).
    pub fn stop_and_join(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
        if let Some(handle) = self.handle.take() {
            let _ = handle.join();
        }
    }
}

impl Drop for EngineThread {
    fn drop(&mut self) {
        self.stop_and_join();
    }
}

/// Laço do tick: comandos da borda (máx. [`COMMANDS_PER_TURN`]) → avanço do relógio.
fn run_loop(
    engine: &mut Engine,
    rx: &Receiver<Command>,
    stop: &AtomicBool,
    intervalo: Duration,
    metrics: &Arc<Metrics>,
) {
    let mut proximo = Instant::now() + intervalo;
    let mut ultimo = Instant::now();
    let mut ultimo_log = Instant::now();
    while !stop.load(Ordering::Relaxed) {
        for command in rx.try_iter().take(COMMANDS_PER_TURN) {
            engine.apply(command);
        }
        let agora = Instant::now();
        if agora >= proximo {
            let elapsed =
                u64::try_from(agora.duration_since(ultimo).as_millis()).unwrap_or(u64::MAX);
            ultimo = agora;
            engine.advance(elapsed);
            proximo += intervalo;
            if agora >= proximo {
                // Stall longo (pausa da VM, debugger): não tenta "recuperar" o atraso.
                proximo = agora + intervalo;
            }
        } else {
            thread::sleep((proximo - agora).min(MAX_SLEEP));
        }
        if ultimo_log.elapsed() >= LOG_EVERY {
            ultimo_log = Instant::now();
            let resumo = metrics.tick_snapshot();
            println!(
                "level=info event=tick ticks={} entities={} players={} p50_us={} p99_us={} mean_us={}",
                metrics.ticks(),
                engine.entities(),
                engine.players(),
                resumo.p50,
                resumo.p99,
                resumo.mean
            );
        }
    }
}
