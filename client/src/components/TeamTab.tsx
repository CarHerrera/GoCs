import React, { useState, useMemo, useEffect, type ReactElement } from 'react';
import StatGrid, { createStatCol, type StatColumn } from '../helpers/StatGrid';

interface RosterPlayer {
  id: number;
  name: string;
  matches: number;
  kills: number;
  deaths: number;
  assists: number;
  adr: number;
  dmg: number;
  hs: number;
  kd: number;
  kast: number;
  rating: number;
  clutchWon: number;
  onek: number;
  twok: number;
  threek: number;
  fourk: number;
  ace: number;
  utilDmg: number;
  current: boolean;
  role: string;
}

interface TeamStats {
  tRounds: number;
  ctRounds: number;
  tWins: number;
  ctWins: number;
  pistolPct: number;
  fullPct: number;
}

interface TeamSummary {
  wins: number;
  loss: number;
  players: RosterPlayer[];
  teamStats: TeamStats;
}

interface RosterRow extends RosterPlayer {
  initials: string;
  kdClass: string;
  ratingClass: string;
  bg: string;
}

type SortKey = keyof RosterPlayer;
type SortDir = 'asc' | 'desc';

const getInitials = (name: string): string => {
  const clean = name.replace(/[^a-zA-Z0-9]/g, '');
  return clean.substring(0, 2).toUpperCase();
};

const getRowBg = (idx: number): string =>
  idx % 2 === 1 ? 'rgba(255,255,255,0.02)' : 'transparent';

// Maps column header labels to the RosterPlayer key they sort on
const SORT_KEYS: Record<string, SortKey> = {
  Player:   'name',
  Matches:  'matches',
  Kills:    'kills',
  Assists:  'assists',
  Deaths:   'deaths',
  'KD+-':   'kills',   // diff is derived, sort by kills as proxy; swap to a derived field if you add kd_diff
  'K / D':  'kd',
  '1K':     'onek',
  '2K':     'twok',
  '3K':     'threek',
  '4K':     'fourk',
  '5K':     'ace',
  ADR:      'adr',
  DMG:      'dmg',
  UD:       'utilDmg',
  'HS %':   'hs',
  'KAST%':  'kast',
  Rating:   'rating',
};

const COLUMNS = Object.keys(SORT_KEYS);

const TeamTab: React.FC = (): ReactElement => {
  const [data, setData]       = useState<TeamSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [sortKey, setSortKey] = useState<SortKey>('rating');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  useEffect(() => {
    fetch('http://localhost:4000/api/team/summary', { credentials: 'include' })
      .then(res => {
        if (res.status === 401) { window.location.href = '/'; return null; }
        if (!res.ok) throw new Error('Failed to load team summary');
        return res.json();
      })
      .then((d: TeamSummary | null) => { if (d) setData(d); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleSort = (col: string): void => {
    const key = SORT_KEYS[col];
    if (!key) return;
    if (key === sortKey) {
      // Same column — flip direction
      setSortDir(d => d === 'desc' ? 'asc' : 'desc');
    } else {
      // New column — default to descending so best values appear first
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const sortIndicator = (col: string): string => {
    if (SORT_KEYS[col] !== sortKey) return '';
    return sortDir === 'desc' ? ' ▼' : ' ▲';
  };

  const teamStats: StatColumn[] = useMemo(() => {
    if (!data) return [];
    const total = data.wins + data.loss;
    const ts = data.teamStats;
    return [
      createStatCol('Matches',      total, 0, false),
      createStatCol('Win %',        `${Math.round((data.wins / total) * 100)}%`, 1, true),
      createStatCol('Pistol Win %', `${ts.pistolPct}%`, 2, false),
      createStatCol('Full Buy %',   `${ts.fullPct}%`, 3, true),
      createStatCol('T Win %',      `${Math.round((ts.tWins / ts.tRounds) * 100)}%`, 4, false),
      createStatCol('CT Win %',     `${Math.round((ts.ctWins / ts.ctRounds) * 100)}%`, 5, true),
    ];
  }, [data]);

  const roster: RosterRow[] = useMemo(() => {
    if (!data) return [];

    const sorted = [...data.players].sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (typeof av === 'string' && typeof bv === 'string') {
        return sortDir === 'asc'
          ? av.localeCompare(bv)
          : bv.localeCompare(av);
      }
      const an = Number(av);
      const bn = Number(bv);
      return sortDir === 'asc' ? an - bn : bn - an;
    });

    return sorted.map((p, idx): RosterRow => ({
      ...p,
      initials:    getInitials(p.name),
      kdClass:     p.kd >= 1 ? 'cell-kd-good' : 'cell-kd-bad',
      ratingClass: p.rating >= 1.05 ? 'cell-rating-good' : p.rating < 1 ? 'cell-rating-bad' : 'cell-rating-neutral',
      bg:          getRowBg(idx),
    }));
  }, [data, sortKey, sortDir]);

  if (loading) return <div className="loading">Loading team data...</div>;
  if (!data)   return <div className="loading">No team data found.</div>;
  console.log(data)
  return (
    <>
      <div className="stat-grid-container">
        <div className="stat-grid-label">Team — Career</div>
        <StatGrid stats={teamStats} />
      </div>

      <div className="roster-container">
        <div className="roster-header">
          <div className="roster-label">Roster — All Players</div>
          <span className="roster-hint">Click a column header to sort</span>
        </div>
        <div className="table-wrapper">
          <table className="roster-table">
            <thead>
              <tr>
                {COLUMNS.map(col => (
                  <th
                    key={col}
                    onClick={() => handleSort(col)}
                    className={`sortable-header ${SORT_KEYS[col] === sortKey ? 'sort-active' : ''}`}
                  >
                    {col}{sortIndicator(col)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roster.map((p, idx): ReactElement => (
                <tr key={p.id ?? idx} style={{ backgroundColor: p.bg }}>
                  <td>
                    <div className="player-cell">
                      <div className="player-avatar">{p.initials}</div>
                        <div>
                          <div className="player-info-name">{p.name}</div>
                          {p.role && <div className="player-info-role">{p.role}</div>}
                        </div>
                    </div>
                  </td>
                  <td className="cell-matches">{p.matches}</td>
                  <td className="cell-kills">{p.kills}</td>
                  <td className="cell-kills">{p.assists}</td>
                  <td className="cell-kills">{p.deaths}</td>
                  <td className="cell-adr">{p.kills - p.deaths}</td>
                  <td className={`cell-kd ${p.kdClass}`}>{p.kd}</td>
                  <td className="cell-adr">{p.onek}</td>
                  <td className="cell-adr">{p.twok}</td>
                  <td className="cell-adr">{p.threek}</td>
                  <td className="cell-adr">{p.fourk}</td>
                  <td className="cell-adr">{p.ace}</td>
                  <td className="cell-adr">{p.adr}</td>
                  <td className="cell-adr">{p.dmg}</td>
                  <td className="cell-adr">{p.utilDmg}</td>
                  <td className="cell-hs">{p.hs}%</td>
                  <td className="cell-kast">{p.kast}%</td>
                  <td className={`cell-rating ${p.ratingClass}`}>{p.rating}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="matches-container">
        <div className="matches-label">Recent Matches</div>
        <div className="matches-list"></div>
      </div>
    </>
  );
};

export default TeamTab;