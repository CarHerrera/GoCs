import React, { useState, useEffect, useRef, type ReactElement } from 'react';
import '../css/SettingsModal.css';
import UploadDemoModal from './UploadDemoModal';

interface PlayerRole {
  id: string;
  name: string;
  role: string;
}

interface UploadedDemo {
  demoName: string;
  map: string;
  date: string;
  season: string;
  notes: string;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
  teamName: string;
}

interface RosterMember {
  id: string;          // string for Steam ID precision
  name: string;
  avatarUrl: string;
  isActive: boolean;
  rosterOrder: number;
}

const RosterSection: React.FC = () => {
  const [members, setMembers] = useState<RosterMember[]>([]);
  const [inviteSteamId, setInviteSteamId] = useState('');
  const [inviting, setInviting] = useState(false);
  const dragItem = useRef<number | null>(null);

  useEffect(() => {
    fetch('http://localhost:4000/api/team/roster', { credentials: 'include' })
      .then(r => r.json()).then(setMembers).catch(() => {});
  }, []);

  const handleDragStart = (idx: number) => { dragItem.current = idx; };

  const handleDragEnter = (idx: number) => {
    if (dragItem.current === null) return;
    const next = [...members];
    const dragged = next.splice(dragItem.current, 1)[0];
    next.splice(idx, 0, dragged);
    dragItem.current = idx;
    setMembers(next);
  };

  const handleDragEnd = async () => {
    const ordered = members.map((m, idx) => ({ id: m.id, order: idx }));
    await fetch('http://localhost:4000/api/team/roster/order', {
      method: 'PUT', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(ordered),
    });
    dragItem.current = null;
  };

  const toggleActive = async (id: string, current: boolean) => {
    setMembers(ms => ms.map(m => m.id === id ? { ...m, isActive: !current } : m));
    await fetch(`http://localhost:4000/api/player/${id}/active`, {
      method: 'PUT', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isActive: !current }),
    });
  };

  const sendInvite = async () => {
    if (!inviteSteamId.trim()) return;
    setInviting(true);
    await fetch('http://localhost:4000/api/team/invite', {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ steamId: inviteSteamId }),
    });
    setInviteSteamId('');
    setInviting(false);
  };

  return (
    <div className="settings-section">
      <div className="settings-field">
        <label>Invite player by Steam ID</label>
        <div className="settings-invite-row">
          <input className="settings-input" placeholder="7656119..."
            value={inviteSteamId} onChange={e => setInviteSteamId(e.target.value)} />
          <button className="settings-save-btn" onClick={sendInvite} disabled={inviting}>
            {inviting ? '…' : 'Invite'}
          </button>
        </div>
      </div>

      <p className="settings-section-desc">Drag to reorder · Toggle active status</p>
      <div className="settings-roster-list">
        {members.map((m, idx) => (
          <div key={m.id} className="settings-roster-row" draggable
            onDragStart={() => handleDragStart(idx)}
            onDragEnter={() => handleDragEnter(idx)}
            onDragEnd={handleDragEnd}
            onDragOver={e => e.preventDefault()}>
            <span className="drag-handle">⠿</span>
            <img src={m.avatarUrl || '/default-avatar.png'}
              className="settings-player-avatar-img" alt={m.name} />
            <span className="settings-player-name">{m.name}</span>
            <button
              className={`active-toggle ${m.isActive ? 'active-toggle-on' : 'active-toggle-off'}`}
              onClick={() => toggleActive(m.id, m.isActive)}>
              {m.isActive ? 'Active' : 'Sub'}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
type Section = 'team' | 'roster' | 'players' | 'seasons' | 'files';

const SettingsModal: React.FC<Props> = ({ isOpen, onClose, teamName }): ReactElement => {
  const [section, setSection] = useState<Section>('team');
  const [players, setPlayers] = useState<PlayerRole[]>([]);
  const [roles, setRoles] = useState<Record<string, string>>({});
  const [savingRoles, setSavingRoles] = useState<Record<string, boolean>>({});
  const [demos, setDemos] = useState<UploadedDemo[]>([]);
  const [loadingDemos, setLoadingDemos] = useState(false);
  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const logoInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isOpen || section !== 'players') return;
    fetch('http://localhost:4000/api/team/Playerstats/', { credentials: 'include' })
      .then(r => r.json())
      .then((data: PlayerRole[]) => {
        setPlayers(data);
        const map: Record<string, string> = {};
        data.forEach(p => { map[p.id] = p.role ?? ''; });
        setRoles(map);
      })
      .catch(() => {});
  }, [isOpen, section]);

  const fetchDemos = () => {
    setLoadingDemos(true);
    fetch('http://localhost:4000/api/team/demos', { credentials: 'include' })
      .then(r => r.json())
      .then(data => setDemos(Array.isArray(data) ? data : []))
      .catch(() => setDemos([]))
      .finally(() => setLoadingDemos(false));
  };

  useEffect(() => {
    if (!isOpen || section !== 'files') return;
    fetchDemos();
  }, [isOpen, section]);

  const handleLogoChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = ev => setLogoPreview(ev.target?.result as string);
    reader.readAsDataURL(file);

    const fd = new FormData();
    fd.append('logo', file);
    await fetch('http://localhost:4000/api/team/logo', {
      method: 'POST',
      credentials: 'include',
      body: fd,
    });
  };

const saveRole = async (playerId: string) => {
  setSavingRoles(s => ({ ...s, [playerId]: true }));
  try {
    await fetch(`http://localhost:4000/api/player/${playerId}/role`, {
      method: 'PUT',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role: roles[playerId] ?? '' }),
    });
  } finally {
    setSavingRoles(s => ({ ...s, [playerId]: false }));
  }
};
  if (!isOpen) return <></>;

  return (
    <>
      <div className="settings-overlay" onClick={onClose}>
        <div className="settings-drawer" onClick={e => e.stopPropagation()}>
          <div className="settings-header">
            <h2>Settings</h2>
            <button className="settings-close" onClick={onClose}>✕</button>
          </div>

          <div className="settings-nav">
            {(['team', 'roster', 'players', 'seasons', 'files'] as Section[]).map(s => (
              <button
                key={s}
                className={`settings-nav-btn ${section === s ? 'active' : ''}`}
                onClick={() => setSection(s)}
              >
                {s === 'team' ? 'Team Info' : s === 'players' ? 'Player Roles' : 'Files'}
              </button>
            ))}
          </div>

          <div className="settings-body">
            {section === 'team' && (
              <div className="settings-section">
                <div className="settings-field">
                  <label>Team Name</label>
                  <input
                    className="settings-input"
                    value={teamName}
                    readOnly
                  />
                  <span className="settings-hint">Team name is derived from your match data</span>
                </div>

                <div className="settings-field">
                  <label>Team Logo</label>
                  <div className="logo-upload-row">
                    <div className="logo-preview">
                      {logoPreview ? (
                        <img src={logoPreview} alt="Team logo" />
                      ) : (
                        <span>{teamName?.slice(0, 2).toUpperCase() || 'RV'}</span>
                      )}
                    </div>
                    <button
                      className="settings-upload-btn"
                      onClick={() => logoInputRef.current?.click()}
                    >
                      Upload Logo
                    </button>
                    <input
                      ref={logoInputRef}
                      type="file"
                      accept="image/*"
                      style={{ display: 'none' }}
                      onChange={handleLogoChange}
                    />
                  </div>
                </div>
              </div>
            )}

            {section === 'players' && (
              <div className="settings-section">
                <p className="settings-section-desc">Assign in-game roles to your roster.</p>
                {players.length === 0 ? (
                  <p className="settings-empty">No players found.</p>
                ) : (
                  players.map(p => (
                    <div key={p.id} className="settings-player-row">
                      <div className="settings-player-avatar">
                        {p.name.replace(/[^a-zA-Z0-9]/g, '').substring(0, 2).toUpperCase()}
                      </div>
                      <span className="settings-player-name">{p.name}</span>
                      <input
                        className="settings-role-input"
                        placeholder="IGL, AWPer, Entry…"
                        value={roles[p.id] ?? ''}
                        onChange={e => setRoles(r => ({ ...r, [p.id]: e.target.value }))}
                        onKeyDown={e => { if (e.key === 'Enter') saveRole(p.id); }}
                      />
                      <button
                        className="settings-save-btn"
                        onClick={() => saveRole(p.id)}
                        disabled={!!savingRoles[p.id]}
                      >
                        {savingRoles[p.id] ? '…' : 'Save'}
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}

            {section === 'files' && (
              <div className="settings-section">
                <div className="settings-files-header">
                  <p className="settings-section-desc">Uploaded demo files.</p>
                  <button
                    className="settings-upload-btn"
                    onClick={() => setShowUpload(true)}
                  >
                    + Upload Demo
                  </button>
                </div>

                {loadingDemos ? (
                  <p className="settings-loading">Loading…</p>
                ) : demos.length === 0 ? (
                  <p className="settings-empty">No demos uploaded yet.</p>
                ) : (
                  <div className="settings-file-list">
                    {demos.map(d => (
                      <div key={d.demoName} className="settings-file-item">
                        <div className="settings-file-info">
                          <span className="settings-file-name">{d.demoName}</span>
                          <span className="settings-file-meta">
                            {d.map}{d.date ? ` · ${d.date}` : ''}
                          </span>
                          {d.season && (
                            <span className="settings-file-season">Season: {d.season}</span>
                          )}
                          {d.notes && (
                            <span className="settings-file-notes">{d.notes}</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {showUpload && (
        <UploadDemoModal
          onClose={() => setShowUpload(false)}
          onUploaded={() => {
            setShowUpload(false);
            fetchDemos();
          }}
        />
      )}
    </>
  );
};

export default SettingsModal;
