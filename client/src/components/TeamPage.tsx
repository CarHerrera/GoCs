import React, { useState, type ReactElement } from 'react';
import '../css/TeamStatsDashboard.css';
import TeamTab from './TeamTab';
import PlayersTab from './PlayersTab';
import AdvancedTab from './AdvancedTab';
import SettingsModal from './SettingsModal';
import UploadDemoModal from './UploadDemoModal';
import { useAuth } from '../context/AuthContext';

type TabType = 'Team' | 'Players' | 'Advanced';

const TeamStatsDashboard: React.FC = (): ReactElement => {
  const [tab, setTab] = useState<TabType>('Team');
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const { user, loading } = useAuth();
  return (
    <div className="dashboard">
      <div className="dashboard-container">
        {/* Team Header */}
        <div className="header">
          <div className="header-logo">RV</div>
          <div className="header-info">
            <h1>{loading ? (
                ""
              ) :  user ? (
                  user.teamName
              ) : ( "Error"

              ) }</h1>
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
        teamName={user?.teamName ?? ''}
      />
    </div>
  );
};

export default TeamStatsDashboard;
