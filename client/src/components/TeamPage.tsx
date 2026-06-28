import React, { useState, useMemo, type ReactElement, useEffect } from 'react';
import './TeamStatsDashboard.css';

// ============================================================================
// TYPES
// ============================================================================
interface Team {
  wins: number
  loss: number
  players: Player[]
  pistolpct: number;
  ecopct: number; 
  fullpct: number;
  forcepct: number;
}
interface Player {
  name: string;
  role: string;
  matches: number;
  kills: number;
  deaths: number;
  dmg: number;
  assists: number;
  adr: number;
  hs: number;
  kast: number;
  rating: number;
  clutchWon: number;
  clutchPct: number;
  entryKills: number;
  onek: number;
  twok: number;
  threek: number;
  fourk: number;
  ace: number;
  openingPct: number;
  tradePct: number;
  utilDmg: number;
  current: boolean;
}

interface RosterPlayer extends Player {
  initials: string;
  kd: string;
  kdColor: string;
  adr: number;
  hs: number;
  rating: number;
  ratingColor: string;
  bg: string;
}

interface Match {
  result: string;
  win: boolean;
  opponent: string;
  map: string;
  topName: string;
  topLine: string;
}

interface StatColumn {
  label: string;
  value: string | number;
  divider: string;
  valueColor: string;
}

interface MapStats {
  name: string;
  rounds: number;
  tWin: number;
  ctWin: number;
  pistol: number;
  tColor: string;
  ctColor: string;
  pistolColor: string;
  bg: string;
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

type TabType = 'Team' | 'Players' | 'Advanced';

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

const getInitials = (name: string): string => {
  const clean = name.replace(/[^a-zA-Z0-9]/g, '');
  return clean.substring(0, 2).toUpperCase();
};

const getRowBg = (idx: number): string => {
  return (idx % 2 === 1) ? 'rgba(255,255,255,0.02)' : 'transparent';
};

const createStatCol = (
  label: string,
  value: string | number,
  idx: number,
  highlight: boolean
): StatColumn => ({
  label,
  value,
  divider: idx === 0 ? 'none' : '1px solid rgba(255,255,255,0.06)',
  valueColor: highlight ? '#3b82f6' : '#ffffff',
});

const getPercentColor = (value: number, highThreshold: number, lowThreshold: number): string => {
  if (value >= highThreshold) return '#2ecc71';
  if (value < lowThreshold) return '#e25563';
  return '#ECECF1';
};

// ============================================================================
// MAIN COMPONENT
// ============================================================================

const TeamStatsDashboard: React.FC = (): ReactElement => {
  const [tab, setTab] = useState<TabType>('Team');
  const [selectedPlayer, setSelectedPlayer] = useState<number>(0);
  const [playerData, setPlayerData] = useState<Team | null> (null);

  useEffect(() => {
    fetch(`http://localhost:4000/api/player/team`, {credentials: 'include'})
      .then(res => {
          if (res.status === 401) {
            // Not logged in — kick them back to login
            window.location.href = '/';
            return null;
          }
          if (!res.ok) throw new Error('Failed to load player data');
          return res.json();
        })
      .then((d: Team | null) => {
        if (!d) return;
        setPlayerData(d)
      })
      .catch(err => {
          console.error(err);
      })
  }, [])
  console.log(playerData)
  // ========================================================================
  // DATA - Replace with API calls
  // ========================================================================
  const players: Player[] = [ ];
  const teamStats: StatColumn[] = [];
  
  if (playerData){
    let kills = 0
    let deaths = 0
    playerData.players.forEach((p) => {
      players.push(p)
      kills += p.kills
      deaths += p.deaths
    })
    teamStats.push(createStatCol('Matches', playerData!.wins + playerData!.loss, 0,false))
    teamStats.push(createStatCol('Wins', playerData!.wins, 1, false))
    teamStats.push(createStatCol('Win %', Math.round((playerData!.wins / (playerData!.wins + playerData!.loss)) * 100)/100, 2, false))
    teamStats.push(createStatCol('Team K / D', Math.round(kills/deaths * 100)/100, 3, true),)
  } else {
    players.push(
      { name: 'AsO4-', role: 'AWP', matches: 70, kills: 1225, deaths: 1127, assists: 279, adr: 76.4, hs: 48, kast: 71, rating: 1.12, clutchWon: 38, clutchPct: 41, entryKills: 96, openingPct: 54, tradePct: 63, utilDmg: 8.4, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'v1go', role: 'IGL · Rifle', matches: 70, kills: 1080, deaths: 1090, assists: 360, adr: 71.2, hs: 52, kast: 73, rating: 1.02, clutchWon: 31, clutchPct: 37, entryKills: 64, openingPct: 48, tradePct: 66, utilDmg: 11.2, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'mYst', role: 'Entry', matches: 68, kills: 1190, deaths: 1160, assists: 240, adr: 78.9, hs: 56, kast: 69, rating: 1.08, clutchWon: 22, clutchPct: 33, entryKills: 188, openingPct: 58, tradePct: 60, utilDmg: 6.1, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'Karob', role: 'Support', matches: 70, kills: 940, deaths: 1050, assists: 410, adr: 66.5, hs: 45, kast: 74, rating: 0.96, clutchWon: 26, clutchPct: 35, entryKills: 52, openingPct: 44, tradePct: 70, utilDmg: 14.6, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'refrezh', role: 'Lurk', matches: 70, kills: 1100, deaths: 1010, assists: 280, adr: 75.3, hs: 51, kast: 72, rating: 1.10, clutchWon: 35, clutchPct: 39, entryKills: 78, openingPct: 50, tradePct: 65, utilDmg: 9.2, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'f0xx', role: 'Lurker', matches: 64, kills: 1110, deaths: 1020, assists: 270, adr: 74.1, hs: 50, kast: 70, rating: 1.10, clutchWon: 44, clutchPct: 46, entryKills: 71, openingPct: 51, tradePct: 58, utilDmg: 7.3, current: true, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'nyte', role: 'Rifle · Sub', matches: 22, kills: 360, deaths: 350, assists: 90, adr: 72.0, hs: 49, kast: 70, rating: 1.04, clutchWon: 9, clutchPct: 38, entryKills: 40, openingPct: 52, tradePct: 62, utilDmg: 8.0, current: false, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
      { name: 'zede', role: 'Rifle · Former', matches: 18, kills: 280, deaths: 300, assists: 70, adr: 68.4, hs: 47, kast: 68, rating: 0.95, clutchWon: 6, clutchPct: 31, entryKills: 33, openingPct: 47, tradePct: 61, utilDmg: 7.9, current: false, dmg: 100, onek: 100, twok: 200, threek:300, fourk: 400, ace:500},
    )
    teamStats.push(
      createStatCol('Matches', '70', 0, false),
      createStatCol('Wins', '34', 1, false),
      createStatCol('Win %', '49%', 2, false),
      createStatCol('Round Win %', '51%', 3, false),
      createStatCol('Team K / D', '1.02', 4, true),
      createStatCol('Pistol Win %', '53%', 5, false),
    )
  }
  



  // const recentMatches: Match[] = [
  //   { result: 'W 13-8', win: true, opponent: 'Spectrum', map: 'de_ancient', topName: 'AsO4-', topLine: '24-14' },
  //   { result: 'L 9-13', win: false, opponent: 'SG-Fusion', map: 'de_overpass', topName: 'mYst', topLine: '21-17' },
  //   { result: 'L 8-13', win: false, opponent: 'Adroit 5', map: 'de_ancient', topName: 'f0xx', topLine: '19-16' },
  //   { result: 'W 13-10', win: true, opponent: 'MonkeyGaming', map: 'de_dust2', topName: 'AsO4-', topLine: '26-13' },
  //   { result: 'L 6-13', win: false, opponent: 'Les poules en rute', map: 'de_mirage', topName: 'mYst', topLine: '18-15' },
  // ];

  // const economyStats: StatColumn[] = [
  //   createStatCol('Pistol Win %', '53%', 0, true),
  //   createStatCol('T-Side Win %', '49%', 1, false),
  //   createStatCol('CT-Side Win %', '56%', 2, false),
  //   createStatCol('Eco Conv %', '24%', 3, false),
  //   createStatCol('Force-Buy %', '42%', 4, false),
  //   createStatCol('Full-Buy %', '67%', 5, false),
  // ];

  // ========================================================================
  // COMPUTED DATA
  // ========================================================================

  const roster = useMemo((): RosterPlayer[] => {
    return players.map((p, idx): RosterPlayer => ({
      ...p,
      initials: getInitials(p.name),
      kd: (p.kills / p.deaths).toFixed(2),
      kdColor: (p.kills / p.deaths) >= 1 ? '#2ecc71' : '#e25563',
      adr: Number(p.adr.toFixed(1)),
      hs: p.hs,
      rating: Number(p.rating.toFixed(2)),
      ratingColor: p.rating >= 1.05 ? '#2ecc71' : (p.rating < 1 ? '#e25563' : '#ECECF1'),
      bg: getRowBg(idx),
    }));
  }, [playerData]);

  const playerIdx: number = Math.min(selectedPlayer, players.length - 1);
  const selectedPlayerData: Player = players[playerIdx];
  const rounds: number = selectedPlayerData.matches * 24;

  const playerButtons = useMemo((): PlayerButton[] => {
    return players.map((p, idx): PlayerButton => ({
      name: p.name,
      role: p.role,
      bg: idx === playerIdx ? 'rgba(59, 130, 246, 0.16)' : 'rgba(255,255,255,0.03)',
      border: idx === playerIdx ? 'rgba(59, 130, 246, 0.3)' : 'rgba(255,255,255,0.06)',
      nameColor: idx === playerIdx ? '#3b82f6' : '#ECECF1',
      onClick: (): void => setSelectedPlayer(idx),
      active: idx === playerIdx,
    }));
  }, [playerIdx]);

  const playerCombat: StatColumn[] = [
    createStatCol('K / D', (selectedPlayerData.kills / selectedPlayerData.deaths).toFixed(2), 0, true),
    createStatCol('ADR', selectedPlayerData.adr.toFixed(1), 1, false),
    createStatCol('KAST %', `${selectedPlayerData.kast}%`, 2, false),
    createStatCol('HS %', `${selectedPlayerData.hs}%`, 3, false),
    createStatCol('KPR', (selectedPlayerData.kills / rounds).toFixed(2), 4, false),
    createStatCol('DPR', (selectedPlayerData.deaths / rounds).toFixed(2), 5, false),
  ];

  const playerClutch: StatColumn[] = [
    createStatCol('1vX Won', `${selectedPlayerData.clutchWon}`, 0, false),
    createStatCol('Clutch %', `${selectedPlayerData.clutchPct}%`, 1, true),
    createStatCol('Entry Kills', `${selectedPlayerData.entryKills}`, 2, false),
    createStatCol('Opening %', `${selectedPlayerData.openingPct}%`, 3, false),
    createStatCol('Trade %', `${selectedPlayerData.tradePct}%`, 4, false),
    createStatCol('Util Dmg/rd', selectedPlayerData.utilDmg.toFixed(1), 5, false),
  ];

  const maps = useMemo((): MapStats[] => {
    const rawMaps: Array<Omit<MapStats, 'tColor' | 'ctColor' | 'pistolColor' | 'bg'>> = [
      { name: 'de_ancient', rounds: 540, tWin: 52, ctWin: 56, pistol: 58 },
      { name: 'de_inferno', rounds: 230, tWin: 49, ctWin: 60, pistol: 55 },
      { name: 'de_mirage', rounds: 320, tWin: 54, ctWin: 51, pistol: 52 },
      { name: 'de_dust2', rounds: 350, tWin: 47, ctWin: 58, pistol: 50 },
      { name: 'de_overpass', rounds: 300, tWin: 44, ctWin: 53, pistol: 46 },
      { name: 'de_nuke', rounds: 180, tWin: 41, ctWin: 62, pistol: 49 },
    ];
    return rawMaps.map((m, idx): MapStats => ({
      name: m.name,
      rounds: m.rounds,
      tWin: m.tWin,
      ctWin: m.ctWin,
      pistol: m.pistol,
      tColor: getPercentColor(m.tWin, 52, 46),
      ctColor: getPercentColor(m.ctWin, 55, 50),
      pistolColor: getPercentColor(m.pistol, 53, 48),
      bg: getRowBg(idx),
    }));
  }, []);

  // ========================================================================
  // STAT GRID COMPONENT
  // ========================================================================

  interface StatGridProps {
    stats: StatColumn[];
  }

  const StatGrid: React.FC<StatGridProps> = ({ stats }): ReactElement => (
    <div className="stat-grid">
      {stats.map((stat, idx): ReactElement => (
        <div
          key={idx}
          className="stat-column"
          style={{ borderLeft: stat.divider }}
        >
          <div className={`stat-column-value ${stat.valueColor === '#3b82f6' ? 'highlight' : ''}`}>
            {stat.value}
          </div>
          <div className="stat-column-label">{stat.label}</div>
        </div>
      ))}
    </div>
  );

  // ========================================================================
  // RENDER
  // ========================================================================

  return (
    <div className="dashboard">
      {/* Navigation */}
      <nav className="nav">
        <div className="nav-bar">
          {['Home', 'Matches', 'StratLab', 'Logout'].map((label: string, idx: number): ReactElement => (
            <button key={idx} className="nav-button">
              {label}
            </button>
          ))}
        </div>
      </nav>

      <div className="dashboard-container">
        {/* Team Header */}
        <div className="header">
          <div className="header-logo">RV</div>
          <div className="header-info">
            <h1>Rampant Veggies</h1>
            <div className="header-info-subtitle">
              <span className="header-info-text">Counter-Strike 2</span>
              <span className="header-divider"></span>
              <span className="header-meta">7 players · 70 matches · 5 seasons</span>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="tabs-container">
          {(['Team', 'Players', 'Advanced'] as const).map((t: TabType): ReactElement => (
            <button
              key={t}
              className={`tab-button ${tab === t ? 'active' : ''}`}
              onClick={(): void => setTab(t)}
            >
              {t}
            </button>
          ))}
        </div>
        
        {/* TEAM TAB */}
        {tab === 'Team' && (
          <>
            {/* Team Career Stats */}
            <div className="stat-grid-container">
              <div className="stat-grid-label">Team — Career</div>
              <StatGrid stats={teamStats} />
            </div>

            {/* Roster Table */}
            <div className="roster-container">
              <div className="roster-header">
                <div className="roster-label">Roster — All Players</div>
                <span className="roster-hint">Click a player for full stats</span>
              </div>
              <div className="table-wrapper">
                <table className="roster-table">
                  <thead>
                    <tr>
                      <th>Player</th>
                      <th>Matches</th>
                      <th>Kills</th>
                      <th>Assists</th>
                      <th>Deaths</th>
                      <th>KD+-</th>
                      <th>K / D</th>
                      <th>1K</th>
                      <th>2K</th>
                      <th>3K</th>
                      <th>4K</th>
                      <th>5K</th>
                      <th>ADR</th>
                      <th>DMG</th>
                      <th>UD</th>
                      <th>HS %</th>
                      <th>KAST%</th>
                      <th>Rating</th>
                    </tr>
                  </thead>
                  <tbody>
                    {roster.map((p: RosterPlayer, idx: number): ReactElement => (
                      <tr
                        key={idx}
                        onClick={(): void => setSelectedPlayer(idx)}
                        style={{ backgroundColor: p.bg }}
                      >
                        <td>
                          <div className="player-cell">
                            <div className="player-avatar">{p.initials}</div>
                            <div>
                              <div className="player-info-name">{p.name}</div>
                              <div className="player-info-role">{p.role}</div>
                            </div>
                          </div>
                        </td>
                        <td className="cell-matches">{p.matches}</td>
                        <td className="cell-kills">{p.kills}</td>
                        <td className="cell-ass">{p.assists}</td>
                        <td className="cell-deaths">{p.deaths}</td>
                        <td className="cell-deaths">{p.kills - p.deaths}</td>
                        <td className={`cell-kd ${p.kdColor === '#2ecc71' ? 'cell-kd-good' : 'cell-kd-bad'}`}>
                          {p.kd}
                        </td>
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
                        <td
                          className={`cell-rating ${
                            p.ratingColor === '#2ecc71'
                              ? 'cell-rating-good'
                              : p.ratingColor === '#e25563'
                              ? 'cell-rating-bad'
                              : 'cell-rating-neutral'
                          }`}
                        >
                          {p.rating}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Recent Matches */}
            <div className="matches-container">
              <div className="matches-label">Recent Matches</div>
              <div className="matches-list">
                {/* {recentMatches.map((m: Match, idx: number): ReactElement => (
                  <div key={idx} className="match-item">
                    <div className={`match-result ${m.win ? 'match-result-win' : 'match-result-loss'}`}>
                      {m.result}
                    </div>
                    <div className="match-opponent">{m.opponent}</div>
                    <div className="match-map">{m.map}</div>
                    <div className="match-top">
                      Top: <span className="match-top-name">{m.topName}</span> {m.topLine}
                    </div>
                  </div>
                ))} */}
              </div>
            </div>
          </>
        )}

        {/* PLAYERS TAB */}
        {tab === 'Players' && (
          <>
            {/* Player Selection Buttons */}
            <div className="player-buttons">
              {playerButtons.map((pb: PlayerButton, idx: number): ReactElement => (
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

            {/* Selected Player Header */}
            <div className="selected-player-header">
              <div className="selected-player-avatar">{getInitials(selectedPlayerData.name)}</div>
              <div className="selected-player-info">
                <div className="selected-player-name-row">
                  <h2>{selectedPlayerData.name}</h2>
                  {selectedPlayerData.current && <span className="active-badge">Active</span>}
                </div>
                <div className="selected-player-subtitle">
                  {selectedPlayerData.role} · {selectedPlayerData.matches} matches
                </div>
              </div>
            </div>

            {/* Combat Profile */}
            <div className="stat-grid-container">
              <div className="stat-grid-label">Combat Profile</div>
              <StatGrid stats={playerCombat} />
            </div>

            {/* Clutch & Entry Fragging */}
            <div className="stat-grid-container">
              <div className="stat-grid-label">Clutch &amp; Entry Fragging</div>
              <StatGrid stats={playerClutch} />
            </div>
          </>
        )}

        {/* ADVANCED TAB */}
        {tab === 'Advanced' && (
          <>
            {/* Round Economy */}
            <div className="stat-grid-container">
              <div className="stat-grid-label">Round Economy</div>
              {/* <StatGrid stats={economyStats} /> */}
            </div>

            {/* Map Performance */}
            <div className="roster-container">
              <div className="stat-grid-label">Side &amp; Pistol Splits by Map</div>
              <p className="maps-description">Round-win rates split by side and pistol rounds</p>
              <div className="table-wrapper">
                <table className="maps-table">
                  <thead>
                    <tr>
                      <th>Map</th>
                      <th>Rounds</th>
                      <th>T Win %</th>
                      <th>CT Win %</th>
                      <th>Pistol Win %</th>
                    </tr>
                  </thead>
                  <tbody>
                    {maps.map((m: MapStats, idx: number): ReactElement => (
                      <tr key={idx} style={{ backgroundColor: m.bg }}>
                        <td className="map-name">{m.name}</td>
                        <td className="map-rounds">{m.rounds}</td>
                        <td className={`map-winrate ${m.tColor === '#2ecc71' ? 'map-winrate-good' : m.tColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                          {m.tWin}%
                        </td>
                        <td className={`map-winrate ${m.ctColor === '#2ecc71' ? 'map-winrate-good' : m.ctColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                          {m.ctWin}%
                        </td>
                        <td className={`map-winrate ${m.pistolColor === '#2ecc71' ? 'map-winrate-good' : m.pistolColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                          {m.pistol}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TeamStatsDashboard;