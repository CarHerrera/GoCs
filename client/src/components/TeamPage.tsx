import React, { useState, useEffect, type ReactElement } from 'react';
import '../css/TeamStatsDashboard.css';
import TeamTab from './TeamTab';
import PlayersTab from './PlayersTab';
import AdvancedTab from './AdvancedTab';
import SettingsModal from './SettingsModal';
import UploadDemoModal from './UploadDemoModal';
import { useAuth } from '../context/AuthContext';

type TabType = 'Team' | 'Players' | 'Advanced';
interface TeamInfo {
  teamName: string;
  playerCount: number;
  matchCount: number;
  seasonCount: number;
}
const TeamStatsDashboard: React.FC = (): ReactElement => {
  const [tab, setTab] = useState<TabType>('Team');
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [teamInfo, setTeamInfo] = useState<TeamInfo | null>(null);
  const fetchTeamInfo = () => {
    fetch('http://localhost:4000/api/team/info', { credentials: 'include' })
      .then(r => r.json()).then(setTeamInfo).catch(() => {});
  };
  useEffect(() => { fetchTeamInfo(); }, []);
  return (
    <div className="dashboard">
      <div className="dashboard-container">
        {/* Team Header */}
        <div className="header">
          <div className="header-logo">RV</div>
          <div className="header-info">
            <h1>{teamInfo?.teamName ?? ""}</h1>
            <span className="header-divider"></span>
            <span className="header-meta">
              {teamInfo?.playerCount ?? 0} Players · {teamInfo?.matchCount ?? 0} Matches · {teamInfo?.seasonCount ?? 0} Seasons
            </span>
            <div className="header-info-subtitle">
              <span className="header-info-text">Counter-Strike 2</span>
            </div>
          </div>
        </div>

        <div className="tabs-container">
          <div className="tabs">
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
          <div style={{ display: 'flex', gap: '8px' }}>
            <button className="settingsBtn" onClick={() => setUploadOpen(true)}>
              ↑ Upload Demo
            </button>
            <button className="settingsBtn" onClick={() => setSettingsOpen(true)}>
              ⚙ Settings
            </button>
          </div>
        </div>

        {tab === 'Team' && <TeamTab />}
        {tab === 'Players' && <PlayersTab />}
        {tab === 'Advanced' && <AdvancedTab />}
      </div>

      {uploadOpen && (
        <UploadDemoModal onClose={() => setUploadOpen(false)} />
      )}
      <SettingsModal
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        teamName={teamInfo?.teamName ?? ''}
        onRenamed={fetchTeamInfo}
      />
    </div>
  );
};

export default TeamStatsDashboard;
