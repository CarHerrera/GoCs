import styles from './PlayerPage.module.css';
import { useState, useEffect } from 'react';
import Cookies from 'js-cookie';
import { useNavigate } from 'react-router-dom';
// ─── Types ───────────────────────────────────────────────────────────────────

type MatchResult = 'win' | 'loss' ;

interface Match {
  file_name: string
  date: string;
  map: string;
  opponent: string;
  result: MatchResult;
  score: string;
  kills: number;
  assists: number;
  deaths: number;
}

interface SeasonStats {
  appearances: string;
  kills: string;
  assists: string;
  deaths: string;
  KD: number;
}

interface GameNote {
  matchLabel: string;
  text: string;
}

interface UnparsedNote {
  text: string;
}

// This is what the API returns — matches your Go response shape
interface PlayerPageData {
  username: string;
  steamLinked: boolean;
  hasMatches: boolean;
  steamId?: number;
  playerName?: string;
  teamName?: string;
  profilePic?: string;
  // these get added later once you build out the stats endpoint
  stats?: SeasonStats;
  recentMatches?: Match[];
  gameNotes?: GameNote[];
  unparsedNotes?: UnparsedNote[];
}

// ─── Sub-components ──────────────────────────────────────────────────────────

const resultLabels: Record<MatchResult, string> = {
  win: 'W',
  loss: 'L',
};

function MatchRow({ match }: { match: Match }) {
  const navigate = useNavigate();
  return (
    <div className={styles.matchRow} onClick={()=> {
      console.log('CLICKED')
      navigate(`/advancedStats?file=${match.file_name}&map=${match.map}`)
    }}>
      <span className={`${styles.matchResult} ${styles[match.result]}`}>
        {resultLabels[match.result]} {match.score}
      </span>
      <span className={styles.matchOpponent}>{match.opponent}</span>
      <span className={styles.matchStat}>
        <span className={styles.hi}>{match.map}</span>
      </span>
      <span className={styles.matchStat}>
        <span className={styles.hi}>{match.kills}</span> kills
      </span>
      <span className={styles.matchStat}>
        <span className={styles.hi}>{match.assists}</span> assists
      </span>
      <span className={styles.matchStat}>
        <span className={styles.hi}>{match.deaths}</span> deaths
      </span>
      
    </div>
  );
}

function NoteCard({ note }: { note: GameNote }) {
  return (
    <div className={styles.noteCard}>
      <div className={styles.noteMatch}>{note.matchLabel}</div>
      <div className={styles.noteText}>{note.text}</div>
    </div>
  );
}

function UnparsedCard({ note }: { note: UnparsedNote }) {
  return (
    <div className={styles.unparsedCard}>
      <div className={styles.rawLabel}>Raw · unprocessed</div>
      <div className={styles.rawText}>{note.text}</div>
      <button className={styles.parseBtn}>Parse note →</button>
    </div>
  );
}

// ─── Loading state ────────────────────────────────────────────────────────────

function LoadingState() {
  return (
    <>
      <nav className={styles.nav}>
        <div className={styles.navLinks}>
          <a href="#">Home</a>
          <a href="#">Matches</a>
          <a href="#">StratLab</a>
          <a href="#">Logout</a>
        </div>
        <div className={styles.navAvatar} />
      </nav>
      <div className={styles.navAccent} />
      <main className={styles.page}>
        <div className={styles.playerHeader}>
          {/* Avatar placeholder */}
          <div className={styles.playerAvatar} style={{ background: '#252836' }} />
          <div className={styles.playerMeta}>
            <div style={{ width: 160, height: 20, background: '#252836', borderRadius: 4, marginBottom: 8 }} />
            <div style={{ width: 220, height: 14, background: '#1c1f2b', borderRadius: 4 }} />
          </div>
        </div>
        <div className={styles.statsPanel} style={{ opacity: 0.4 }}>
          <div className={styles.statsPanelHeader}>
            <span className={styles.label}>Season stats</span>
          </div>
          <div className={styles.statGrid}>
            {['Appearances', 'Goals', 'Assists', 'Avg rating', 'Mins played'].map(label => (
              <div key={label} className={styles.statCell}>
                <div className={styles.statValue}>—</div>
                <div className={styles.statLabel}>{label}</div>
              </div>
            ))}
          </div>
        </div>
      </main>
    </>
  );
}

// ─── No matches state ─────────────────────────────────────────────────────────

function NoMatchesState({ username }: { username: string }) {
  // Shows the shell of the page but with an empty state message
  // instead of blank stats — gives the user context on what to do next
  const initials = username.slice(0, 2).toUpperCase();
  return (
    <>
      <nav className={styles.nav}>
        <div className={styles.navLinks}>
          <a href="#">Home</a>
          <a href="#">Matches</a>
          <a href="#">StratLab</a>
          <a href="#">Logout</a>
        </div>
        <div className={styles.navAvatar} />
      </nav>
      <div className={styles.navAccent} />
      <main className={styles.page}>
        <div className={styles.playerHeader}>
          <div className={styles.playerAvatar}>{initials}</div>
          <div className={styles.playerMeta}>
            <h1>{username}</h1>
            <div className={styles.sub}>No matches parsed yet</div>
          </div>
        </div>
        <div className={styles.statsPanel}>
          <div style={{ padding: '32px 0', textAlign: 'center' }}>
            <div style={{ fontSize: 13, color: '#5c6080', marginBottom: 8 }}>No data yet</div>
            <div style={{ fontSize: 12, color: '#3a3f58' }}>
              Upload a demo file to see your stats appear here
            </div>
          </div>
        </div>
      </main>
    </>
  );
}

// ─── Onboarding modal ─────────────────────────────────────────────────────────

type ModalView = 'choice' | 'manual' | 'success';

function OnboardingModal({ onClose }: { onClose: () => void }) {
  const [view, setView] = useState<ModalView>('choice');
  const [steamId, setSteamId] = useState('');
  const [hint, setHint] = useState({ text: '', valid: false });
  const [loading, setLoading] = useState(false);

  function handleInput(val: string) {
    const digits = val.replace(/\D/g, '');
    setSteamId(digits);
    const remaining = 17 - digits.length;
    if (digits.length === 0) {
      setHint({ text: '', valid: false });
    } else if (digits.length < 17) {
      setHint({ text: `${remaining} digit${remaining === 1 ? '' : 's'} remaining`, valid: false });
    } else {
      setHint({ text: '✓ looks valid', valid: true });
    }
  }

  function handleLink() {
    setLoading(true);
    fetch('http://localhost:4000/api/player/link-steam', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ steamId }),
    })
      .then(res => {
        if (res.status === 409) {
          setHint({ text: '✕ already linked to another account', valid: false });
          return;
        }
        if (!res.ok) {
          setHint({ text: '✕ something went wrong, try again', valid: false });
          return;
        }
        Cookies.set('steamLinked', 'true', { expires: 365 });
        setView('success');
      })
      .catch(() => setHint({ text: '✕ network error, try again', valid: false }))
      .finally(() => setLoading(false));
  }

  return (
    <div className={styles.modalOverlay}>
      <div className={styles.modal}>

        {view === 'choice' && (
          <>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
              <span className={styles.label}>Account setup</span>
              <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#5c6080', cursor: 'pointer', fontSize: 18 }}>✕</button>
            </div>
            <h2>Link your Steam account</h2>
            <p>Connect Steam to unlock full match stats, history syncing, and more.</p>
            <div className={styles.modalDivider} />
            <button className={styles.steamPrimaryBtn} onClick={() => window.location.href = 'http://localhost:4000/auth/steam'}>
              Sign in via Steam
            </button>
            <button className={styles.steamSecondaryBtn} onClick={() => setView('manual')}>
              Enter Steam ID manually
            </button>
            <button className={styles.skipBtn} onClick={onClose}>
              Skip for now — I'll link later
            </button>
          </>
        )}

        {view === 'manual' && (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
              <button
                onClick={() => setView('choice')}
                className={styles.steamSecondaryBtn}
                style={{ width: 28, height: 28, padding: 0, justifyContent: 'center', marginBottom: 0, flexShrink: 0 }}
              >←</button>
              <div>
                <span className={styles.label} style={{ display: 'block' }}>Manual entry</span>
                <span style={{ fontSize: 14, fontWeight: 600, color: '#fff' }}>Enter your Steam ID</span>
              </div>
            </div>
            <p>Your Steam ID is a 17-digit number. Find it at <code>steamid.io</code> or in your Steam profile URL.</p>
            <input
              className={styles.steamInput}
              type="text"
              placeholder="76561198XXXXXXXXX"
              maxLength={17}
              value={steamId}
              onChange={e => handleInput(e.target.value)}
            />
            <div className={`${styles.inputHint} ${hint.valid ? styles.valid : ''}`}>{hint.text}</div>
            <button
              className={styles.steamPrimaryBtn}
              onClick={handleLink}
              disabled={!hint.valid || loading}
              style={!hint.valid || loading ? { background: '#2a3550', color: '#5c6080', cursor: 'not-allowed' } : {}}
            >
              {loading ? 'Linking...' : 'Link account'}
            </button>
          </>
        )}

        {view === 'success' && (
          <div style={{ textAlign: 'center', padding: '16px 0' }}>
            <div style={{ width: 48, height: 48, borderRadius: '50%', background: '#0e2a1a', border: '1px solid #1a4430', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 14px', fontSize: 22, color: '#3dca7a' }}>✓</div>
            <h2>Steam account linked</h2>
            <p>Your Steam ID has been saved to your profile.</p>
            <button className={styles.steamSecondaryBtn} style={{ justifyContent: 'center' }} onClick={onClose}>
              Continue to dashboard
            </button>
          </div>
        )}

      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function PlayerPage() {
  const [data, setData] = useState<PlayerPageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    fetch('http://localhost:4000/api/player/me', { credentials: 'include' })
      .then(res => {
        if (res.status === 401) {
          // Not logged in — kick them back to login
          window.location.href = '/';
          return null;
        }
        if (!res.ok) throw new Error('Failed to load player data');
        return res.json();
      })
      .then((d: PlayerPageData | null) => {
        console.log(d)
        if (!d) return;
        setData(d);
        // Modal decision made here, after we have real data from the server
        // Cookie is just a cache — server data is the source of truth
        const isFirstVisit = !Cookies.get('firstVisit');
        console.log(isFirstVisit)
        if (isFirstVisit && !d.steamLinked) {
          setShowModal(true);
        } else {
          Cookies.set('firstVisit', 'false', { expires: 365 });
        }
      })
      .catch(err => {
        console.error(err);
        setError('Something went wrong loading your profile.');
      })
      .finally(() => setLoading(false));
  }, []); // empty array — runs once on mount, which is exactly what we want

  if (loading) return <LoadingState />;

  if (error) return (
    <main className={styles.page} style={{ paddingTop: 80, textAlign: 'center' }}>
      <div style={{ color: '#ca4343', fontSize: 13 }}>{error}</div>
    </main>
  );

  if (!data) return null;
  console.log(data)
  // State 2 — Steam linked but no demos parsed yet
  if (!data.hasMatches) {
    return <>
      {showModal && <OnboardingModal onClose={() => setShowModal(false)} />}    
      <NoMatchesState username={data.username} />
    </>
  }
     

  // State 3 — full data, render the page
  const initials = data.username.slice(0, 2).toUpperCase();

  return (
    <>
      {showModal && <OnboardingModal onClose={() => setShowModal(false)} />}

      <nav className={styles.nav}>
        <div className={styles.navLinks}>
          <a href="#">Home</a>
          <a href="#">Matches</a>
          <a href="#">StratLab</a>
          <a href="#">Logout</a>
        </div>
        <div className={styles.navAvatar}>
          <img src={data.profilePic}></img>
          </div> 
      </nav>
      <div className={styles.navAccent} />

      <main className={styles.page}>

        {/* Player Header */}
        <div className={styles.playerHeader}>
          <div className={styles.playerAvatar}>{
          data.profilePic  == ""  ? initials : <><img src={data.profilePic!}></img></>
          }</div>
          <div className={styles.playerMeta}>
            <h1>{data.playerName ?? data.username}</h1>
            <div className={styles.sub}>
              {data.teamName ?? 'No team'} 
            </div>
          </div>
        </div>

        {/* Stats Panel */}
        <div className={styles.statsPanel}>
          <div className={styles.statsPanelHeader}>
            <span className={styles.label}>Season stats</span>
          </div>
          <div className={styles.statGrid}>
            {(['appearances', 'kills', 'assists', 'deaths', 'KD'] as const).map(key => (
              <div key={key} className={styles.statCell}>
                <div className={`${styles.statValue}`}>
                  {typeof data.stats?.[key] === 'number' ? Math.floor(data.stats[key] * 100) / 100 : data.stats?.[key] ?? '—'}
                </div>
                <div className={styles.statLabel}>
                  {{ appearances: 'Appearances', kills:'kills', assists: 'Assists', deaths:'deaths', KD:"Ratio" }[key]}
                </div>
              </div>
            ))}
          </div>

          <div className={styles.recentLabel}>Recent matches</div>
          <div className={styles.matchList}>
            {(data.recentMatches ?? []).map((match, i) => (
              <MatchRow key={i} match={match} />
            ))}
          </div>
        </div>

        {/* Team Notes */}
        <div className={styles.teamSection}>
          <div className={styles.teamSectionHeader}>
            <div className={styles.teamName}>
              <div className={styles.teamBadge}>
                {data.teamName?.slice(0, 2).toUpperCase() ?? '??'}
              </div>
              {data.teamName ?? 'No team'}
            </div>
          </div>

          <div className={styles.notesGrid}>
            <div className={styles.notesCol}>
              <div className={styles.notesColHeader}>
                <span className={styles.colLabel}>Games with notes</span>
                <span className={styles.countBadge}>{data.gameNotes?.length ?? 0}</span>
              </div>
              {(data.gameNotes ?? []).map((note, i) => (
                <NoteCard key={i} note={note} />
              ))}
            </div>

            <div className={styles.notesCol}>
              <div className={styles.notesColHeader}>
                <span className={styles.colLabel}>Unparsed notes</span>
                <span className={styles.countBadge}>{data.unparsedNotes?.length ?? 0}</span>
              </div>
              {(data.unparsedNotes ?? []).map((note, i) => (
                <UnparsedCard key={i} note={note} />
              ))}
            </div>
          </div>
        </div>

      </main>
    </>
  );
}