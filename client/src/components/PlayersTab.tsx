import React, { useState, useMemo, useEffect, type ReactElement } from 'react';
import StatGrid, { createStatCol, type StatColumn } from '../helpers/StatGrid';

interface PlayerStat {
  id: number;
  name: string;
  role: string;
  matches: number;
  kills: number;
  deaths: number;
  assists: number;
  adr: number;
  hs: number;
  kd: number;
  kast: number;
  rating: number;
  clutchWon: number;
  clutchPct: number;
  entryKills: number;
  openingPct: number;
  tradePct: number;
  utilDmgPerRd: number;
  roundsPlayed: number;
  current: boolean;
}

interface PlayerButton {
  name: string;
  role: string;
  bg: string;
  border: string;
  nameColor: string;
  onClick: () => void;
  active: boolean;
}

const getInitials = (name: string): string => {
  const clean = name.replace(/[^a-zA-Z0-9]/g, '');
  return clean.substring(0, 2).toUpperCase();
};

const PlayersTab: React.FC = (): ReactElement => {
  const [players, setPlayers] = useState<PlayerStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPlayer, setSelectedPlayer] = useState(0);

  useEffect(() => {
    fetch('http://localhost:4000/api/team/Playerstats/', { credentials: 'include' })
      .then(res => {
        if (res.status === 401) { window.location.href = '/'; return null; }
        if (!res.ok) throw new Error('Failed to load player stats');
        return res.json();
      })
      .then((d: PlayerStat[] | null) => { if (d) setPlayers(d); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const playerIdx = Math.min(selectedPlayer, Math.max(players.length - 1, 0));
  const selected = players[playerIdx];

  const playerButtons = useMemo((): PlayerButton[] =>
    players.map((p, idx): PlayerButton => ({
      name: p.name,
      role: p.role,
      bg: idx === playerIdx ? 'rgba(59, 130, 246, 0.16)' : 'rgba(255,255,255,0.03)',
      border: idx === playerIdx ? 'rgba(59, 130, 246, 0.3)' : 'rgba(255,255,255,0.06)',
      nameColor: idx === playerIdx ? '#3b82f6' : '#ECECF1',
      onClick: (): void => setSelectedPlayer(idx),
      active: idx === playerIdx,
    })),
  [playerIdx, players]);

  const playerCombat: StatColumn[] = useMemo(() => {
    if (!selected) return [];
    const rounds = selected.roundsPlayed || 1;
    return [
      createStatCol('K / D', selected.kd.toFixed(2), 0, true),
      createStatCol('ADR', selected.adr.toFixed(1), 1, false),
      createStatCol('KAST %', `${selected.kast}%`, 2, false),
      createStatCol('HS %', `${selected.hs}%`, 3, false),
      createStatCol('KPR', (selected.kills / rounds).toFixed(2), 4, false),
      createStatCol('DPR', (selected.deaths / rounds).toFixed(2), 5, false),
    ];
  }, [selected]);

  const playerClutch: StatColumn[] = useMemo(() => {
    if (!selected) return [];
    return [
      createStatCol('1vX Won', `${selected.clutchWon}`, 0, false),
      createStatCol('Clutch %', `${selected.clutchPct}%`, 1, true),
      createStatCol('Entry Kills', `${selected.entryKills}`, 2, false),
      createStatCol('Opening %', `${selected.openingPct}%`, 3, false),
      createStatCol('Trade %', `${selected.tradePct}%`, 4, false),
      createStatCol('Util Dmg/rd', selected.utilDmgPerRd.toFixed(1), 5, false),
    ];
  }, [selected]);

  if (loading) return <div className="loading">Loading player stats...</div>;
  if (!selected) return <div className="loading">No player data found.</div>;

  return (
    <>
      <div className="player-buttons">
        {playerButtons.map((pb, idx): ReactElement => (
          <button
            key={idx}
            className={`player-button ${pb.active ? 'active' : ''}`}
            onClick={pb.onClick}
          >
            <div className="player-button-name">{pb.name}</div>
            <div className="player-button-role">{pb.role}</div>
          </button>
        ))}
      </div>

      <div className="selected-player-header">
        <div className="selected-player-avatar">{getInitials(selected.name)}</div>
        <div className="selected-player-info">
          <div className="selected-player-name-row">
            <h2>{selected.name}</h2>
            {selected.current && <span className="active-badge">Active</span>}
          </div>
          <div className="selected-player-subtitle">
            {selected.matches} matches { selected.role && `— ${selected.role}` }
          </div>
        </div>
      </div>

      <div className="stat-grid-container">
        <div className="stat-grid-label">Combat Profile</div>
        <StatGrid stats={playerCombat} />
      </div>

      <div className="stat-grid-container">
        <div className="stat-grid-label">Clutch &amp; Entry Fragging</div>
        <StatGrid stats={playerClutch} />
      </div>
    </>
  );
};

export default PlayersTab;
