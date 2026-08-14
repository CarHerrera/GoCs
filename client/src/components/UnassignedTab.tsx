import React, { useState, useEffect } from 'react';

interface UnassignedPlayer {
  playerId: string;
  playerName: string;
}

interface UnassignedGroup {
  matchId: number;
  sideIndex: number;
  defaultName: string;
  demoName: string;
  map: string;
  players: UnassignedPlayer[];
}

interface Props {
  // When set, only shows unassigned sides for this match (used inline on
  // the match page). Omit for the global Settings review queue.
  matchId?: number;
  // Renders nothing at all once there's nothing left to resolve, instead of
  // an empty-state message — appropriate when embedded inline on a page
  // that already has other content (e.g. the match page), not as a
  // standalone Settings tab.
  hideWhenEmpty?: boolean;
}

const UnassignedSection: React.FC<Props> = ({ matchId, hideWhenEmpty }) => {
  const [groups, setGroups] = useState<UnassignedGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [teamNameInputs, setTeamNameInputs] = useState<Record<string, string>>({});
  const [resolving, setResolving] = useState<Record<string, boolean>>({});
  const [dismissing, setDismissing] = useState<Record<string, boolean>>({});

  const fetchUnassigned = () => {
    setLoading(true);
    const url = matchId
      ? `http://localhost:4000/api/team/unassigned?matchId=${matchId}`
      : 'http://localhost:4000/api/team/unassigned';
    fetch(url, { credentials: 'include' })
      .then(r => r.json())
      .then((data: UnassignedGroup[]) => setGroups(Array.isArray(data) ? data : []))
      .catch(() => setGroups([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchUnassigned(); }, [matchId]);

  const keyFor = (g: UnassignedGroup) => `${g.matchId}-${g.sideIndex}`;

  const resolve = async (g: UnassignedGroup) => {
    const key = keyFor(g);
    const teamName = (teamNameInputs[key] ?? '').trim();
    if (!teamName) return;
    setResolving(r => ({ ...r, [key]: true }));
    try {
      await fetch('http://localhost:4000/api/team/unassigned/resolve', {
        method: 'POST', credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ matchId: g.matchId, sideIndex: g.sideIndex, teamName }),
      });
      fetchUnassigned();
    } finally {
      setResolving(r => ({ ...r, [key]: false }));
    }
  };

  const dismiss = async (g: UnassignedGroup) => {
    const key = keyFor(g);
    setDismissing(d => ({ ...d, [key]: true }));
    try {
      await fetch('http://localhost:4000/api/team/unassigned/dismiss', {
        method: 'POST', credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ matchId: g.matchId, sideIndex: g.sideIndex }),
      });
      fetchUnassigned();
    } finally {
      setDismissing(d => ({ ...d, [key]: false }));
    }
  };

  if (hideWhenEmpty && !loading && groups.length === 0) return null;

  return (
    <div className="settings-section">
      <p className="settings-section-desc">
        These demo sides had no clan tag set, and none of the players on them are on an
        existing roster — assign each one to a real team so their stats attach correctly,
        or dismiss it if you don't care to track that side (e.g. random premier opponents).
      </p>

      {loading ? (
        <p className="settings-loading">Loading…</p>
      ) : groups.length === 0 ? (
        <p className="settings-empty">Nothing to resolve.</p>
      ) : (
        <div className="settings-season-list">
          {groups.map(g => {
            const key = keyFor(g);
            return (
              <div
                key={key}
                className="settings-season-item"
                style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}
              >
                <div className="settings-season-info">
                  <span className="settings-season-name">{g.demoName} · {g.map}</span>
                  <span className="settings-season-dates">
                    "{g.defaultName}" — {g.players.map(p => p.playerName).join(', ')}
                  </span>
                </div>
                <div className="settings-invite-row">
                  <input
                    className="settings-input"
                    placeholder="Team name…"
                    value={teamNameInputs[key] ?? ''}
                    onChange={e => setTeamNameInputs(t => ({ ...t, [key]: e.target.value }))}
                  />
                  <button
                    className="settings-save-btn"
                    onClick={() => resolve(g)}
                    disabled={!!resolving[key] || !!dismissing[key] || !(teamNameInputs[key] ?? '').trim()}
                  >
                    {resolving[key] ? '…' : 'Assign'}
                  </button>
                  <button
                    className="settings-cancel-btn"
                    onClick={() => dismiss(g)}
                    disabled={!!resolving[key] || !!dismissing[key]}
                    title="Ignore this side — its stats stay untracked"
                  >
                    {dismissing[key] ? '…' : 'Dismiss'}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default UnassignedSection;
