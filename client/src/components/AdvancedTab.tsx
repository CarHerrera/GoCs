import React, { useState, useEffect, useMemo, type ReactElement } from 'react';
import StatGrid, { createStatCol, type StatColumn } from '../helpers/StatGrid';

interface MapStat {
  map: string;
  rounds: number;
  tWinPct: number;
  ctWinPct: number;
  pistolWinPct: number;
}

interface Economy {
  pistolPct: number;
  tSidePct: number;
  ctSidePct: number;
  ecoPct: number;
  forcePct: number;
  fullPct: number;
}

interface AdvancedResponse {
  economy: Economy;
  maps: MapStat[];
}

interface MapRow extends MapStat {
  tColor: string;
  ctColor: string;
  pistolColor: string;
  bg: string;
}

const getPercentColor = (value: number, highThreshold: number, lowThreshold: number): string => {
  if (value >= highThreshold) return '#2ecc71';
  if (value < lowThreshold) return '#e25563';
  return '#ECECF1';
};

const getRowBg = (idx: number): string =>
  idx % 2 === 1 ? 'rgba(255,255,255,0.02)' : 'transparent';

const AdvancedTab: React.FC = (): ReactElement => {
  const [data, setData] = useState<AdvancedResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('http://localhost:4000/api/team/advanced', { credentials: 'include' })
      .then(res => {
        if (res.status === 401) { window.location.href = '/'; return null; }
        if (!res.ok) throw new Error('Failed to load advanced stats');
        return res.json();
      })
      .then((d: AdvancedResponse | null) => { if (d) setData(d); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const economyStats: StatColumn[] = useMemo(() => {
    if (!data) return [];
    const e = data.economy;
    return [
      createStatCol('Pistol Win %', `${e.pistolPct}%`, 0, true),
      createStatCol('T-Side Win %', `${e.tSidePct}%`, 1, false),
      createStatCol('CT-Side Win %', `${e.ctSidePct}%`, 2, false),
      createStatCol('Eco Conv %', `${e.ecoPct}%`, 3, false),
      createStatCol('Force-Buy %', `${e.forcePct}%`, 4, false),
      createStatCol('Full-Buy %', `${e.fullPct}%`, 5, false),
    ];
  }, [data]);

  const rows: MapRow[] = useMemo(() => {
    if (!data) return [];
    return data.maps.map((m, idx): MapRow => ({
      ...m,
      tColor: getPercentColor(m.tWinPct, 52, 46),
      ctColor: getPercentColor(m.ctWinPct, 55, 50),
      pistolColor: getPercentColor(m.pistolWinPct, 53, 48),
      bg: getRowBg(idx),
    }));
  }, [data]);

  if (loading) return <div className="loading">Loading advanced stats...</div>;
  if (!data) return <div className="loading">No advanced data found.</div>;

  return (
    <>
      <div className="stat-grid-container">
        <div className="stat-grid-label">Round Economy</div>
        <StatGrid stats={economyStats} />
      </div>

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
              {rows.map((m, idx): ReactElement => (
                <tr key={idx} style={{ backgroundColor: m.bg }}>
                  <td className="map-name">{m.map}</td>
                  <td className="map-rounds">{m.rounds}</td>
                  <td className={`map-winrate ${m.tColor === '#2ecc71' ? 'map-winrate-good' : m.tColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                    {m.tWinPct}%
                  </td>
                  <td className={`map-winrate ${m.ctColor === '#2ecc71' ? 'map-winrate-good' : m.ctColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                    {m.ctWinPct}%
                  </td>
                  <td className={`map-winrate ${m.pistolColor === '#2ecc71' ? 'map-winrate-good' : m.pistolColor === '#e25563' ? 'map-winrate-bad' : 'map-winrate-neutral'}`}>
                    {m.pistolWinPct}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
};

export default AdvancedTab;
