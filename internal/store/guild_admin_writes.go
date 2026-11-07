package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Escritas de admin de guild (issue 45, fatia 2). Todas rodam numa única
// transação com sua linha em guild_admin_audit: uma transferência de liderança que
// confirma sem a trilha de auditoria é precisamente o que um log de moderação não
// transferência pela metade (novo master promovido, antigo ainda nível 9) deixaria
// pode permitir, e um
//
// applied=false significa "nada a fazer" (o personagem não é membro, já é master,
// relação ausente); não é erro.

// SetGuildMasterByCharacter promove um membro a nível 9 e rebaixa o
// líder anterior a nível 0, espelhando ambos em `character` para o próximo boot e
// o caminho de save concordarem com guild_member.
func (s *Store) SetGuildMasterByCharacter(ctx context.Context, guildID int32, characterID int64, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var level int
		err := tx.QueryRow(ctx,
			`SELECT guild_level FROM guild_member WHERE guild_id = $1 AND character_id = $2 FOR UPDATE`,
			guildID, characterID).Scan(&level)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild member %d/%d: %w", guildID, characterID, err)
		}
		if level == 9 {
			return nil
		}

		before, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx,
			`UPDATE guild_member SET guild_level = 0, updated_at = now() WHERE guild_id = $1 AND guild_level = 9`,
			guildID); err != nil {
			return fmt.Errorf("store: demote guild %d master: %w", guildID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE guild_member SET guild_level = 9, updated_at = now() WHERE guild_id = $1 AND character_id = $2`,
			guildID, characterID); err != nil {
			return fmt.Errorf("store: promote guild %d member %d: %w", guildID, characterID, err)
		}
		// Espelha em `character`: o caminho de login/save lê guild_level de lá.
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_level = 0 WHERE guild_id = $1 AND guild_level = 9`, guildID); err != nil {
			return fmt.Errorf("store: demote character guild level for %d: %w", guildID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_level = 9 WHERE id = $1 AND guild_id = $2`,
			characterID, guildID); err != nil {
			return fmt.Errorf("store: promote character %d guild level: %w", characterID, err)
		}

		after, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "set_master",
			map[string]any{"character_id": characterID}, before, after, reason)
	})
	return applied, err
}

// RemoveGuildMemberByCharacter descarta uma associação e limpa os campos de guild
// do personagem. O líder é recusado uma camada acima: remover a linha de nível 9
// deixaria uma guild sem ninguém capaz de passá-la adiante.
func (s *Store) RemoveGuildMemberByCharacter(ctx context.Context, guildID int32, characterID int64, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var level int
		err := tx.QueryRow(ctx,
			`SELECT guild_level FROM guild_member WHERE guild_id = $1 AND character_id = $2 FOR UPDATE`,
			guildID, characterID).Scan(&level)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("store: lock guild member %d/%d: %w", guildID, characterID, err)
		}

		before, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM guild_member WHERE guild_id = $1 AND character_id = $2`,
			guildID, characterID); err != nil {
			return fmt.Errorf("store: delete guild member %d/%d: %w", guildID, characterID, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE characters SET guild_id = 0, guild_level = 0 WHERE id = $1 AND guild_id = $2`,
			characterID, guildID); err != nil {
			return fmt.Errorf("store: clear character %d guild: %w", characterID, err)
		}

		after, err := guildMemberSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		applied = true
		return auditGuild(ctx, tx, moderatorID, guildID, "remove_member",
			map[string]any{"character_id": characterID, "guild_level": level}, before, after, reason)
	})
	return applied, err
}

// SetGuildRelationAudited insere uma relação direcionada de aliado/guerra ou a remove
// (kind 0). O par é validado uma camada acima; os CHECKs da própria tabela (kind 1/2,
// guild_id <> target_guild_id) continuam a última linha de defesa.
func (s *Store) SetGuildRelationAudited(ctx context.Context, guildID, targetGuildID int32, kind int, moderatorID int64, reason string) (bool, error) {
	applied := false
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		before, err := guildRelationSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		var errExec error
		if kind == 0 {
			tag, err := tx.Exec(ctx,
				`DELETE FROM guild_relation WHERE guild_id = $1 AND target_guild_id = $2`,
				guildID, targetGuildID)
			if err != nil {
				return fmt.Errorf("store: delete guild relation %d->%d: %w", guildID, targetGuildID, err)
			}
			applied = tag.RowsAffected() > 0
			errExec = nil
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO guild_relation(guild_id, target_guild_id, kind, updated_at)
				VALUES ($1, $2, $3, now())
				ON CONFLICT (guild_id, kind)
				DO UPDATE SET target_guild_id = EXCLUDED.target_guild_id, updated_at = now()`,
				guildID, targetGuildID, kind); err != nil {
				return fmt.Errorf("store: set guild relation %d->%d: %w", guildID, targetGuildID, err)
			}
			applied = true
		}
		if errExec != nil {
			return errExec
		}
		after, err := guildRelationSnapshot(ctx, tx, guildID)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return auditGuild(ctx, tx, moderatorID, guildID, "relation",
			map[string]any{"target_guild_id": targetGuildID, "kind": kind}, before, after, reason)
	})
	return applied, err
}

// guildMemberSnapshot é o antes/depois da auditoria: a associação da guild como a
// tabela a tem, relida dentro da transação.
func guildMemberSnapshot(ctx context.Context, tx pgx.Tx, guildID int32) ([]byte, error) {
	var js []byte
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(m ORDER BY m.guild_level DESC, m.name), '[]'::jsonb)
		  FROM guild_member m WHERE m.guild_id = $1`, guildID).Scan(&js); err != nil {
		return nil, fmt.Errorf("store: snapshot guild %d members: %w", guildID, err)
	}
	return js, nil
}

// guildRelationSnapshot é a mesma ideia para as relações direcionadas de uma guild.
func guildRelationSnapshot(ctx context.Context, tx pgx.Tx, guildID int32) ([]byte, error) {
	var js []byte
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(r ORDER BY r.kind), '[]'::jsonb)
		  FROM guild_relation r WHERE r.guild_id = $1`, guildID).Scan(&js); err != nil {
		return nil, fmt.Errorf("store: snapshot guild %d relations: %w", guildID, err)
	}
	return js, nil
}

// auditGuild anexa uma linha a guild_admin_audit. target e os snapshots de antes/depois são JSONB;
// os snapshots vêm da tabela, então a trilha mostra o estado que foi realmente
// guardado em vez do pedido que foi enviado.
func auditGuild(ctx context.Context, tx pgx.Tx, moderatorID int64, guildID int32, action string, target map[string]any, before, after []byte, reason string) error {
	rawTarget, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("store: marshal guild audit target: %w", err)
	}
	targetJSON := string(rawTarget)
	if _, err := tx.Exec(ctx, `
		INSERT INTO guild_admin_audit (moderator_id, guild_id, action, target, before, after, reason)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7)`,
		moderatorID, guildID, action, targetJSON, jsonText(before), jsonText(after), reason); err != nil {
		return fmt.Errorf("store: audit guild %d %s: %w", guildID, action, err)
	}
	return nil
}
