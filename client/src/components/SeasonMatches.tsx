// SeasonPage.tsx
import { useParams, Link } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';
interface SeasonMatch {
  matchId: number;
  map: string;
  teamA: string; scoreA: number;
  teamB: string; scoreB: number;
  date: string;
  seasonName: string;
}

const SeasonPage: React.FC = () => {
  const { id } = useParams();
  const { user, loading } = useAuth();
  const [seasonName, setSeasonName] = useState('');
  const [matches, setMatches] = useState<SeasonMatch[]>([]);
  const [isLoading, setLoading] = useState(true);
  const navigate = useNavigate();
  useEffect(() => {
    fetch(`http://localhost:4000/api/season/${id}/matches`, { credentials: 'include' })
      .then(r => r.json())
      .then((data: { seasonName: string; matches: SeasonMatch[] }) => { 
        console.log(data)
        setSeasonName(data.seasonName);
        setMatches(data.matches);
       })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [id]);
  console.log(matches)
  if (isLoading) return <div className="loading">Loading season…</div>;

  return (
    <div className="dashboard">
      <div className="dashboard-container">
        <div className="header">
          <div className="header-info">
            <h1>{seasonName}</h1>
            <span className="header-meta">{matches.length} matches</span>
          </div>
        </div>

        <div className="roster-container">
          <div className="table-wrapper">
            <table className="roster-table">
              <thead>
                <tr>
                  <th>Date</th><th>Map</th><th>Match</th><th>Score</th>
                </tr>
              </thead>
              <tbody>
                {matches.map(m => {
                    let home = {
                        team: "",
                        score: 0
                    };
                    let opp = {
                        team: "",
                        score: 0
                    };
                    if (loading) {
                        return "Loading Matches"
                    } else {
                        
                        if (m.teamA === user!.teamName) {
                            home.team = m.teamA
                            home.score = m.scoreA
                            opp.team = m.teamB
                            opp.score = m.scoreB
                        } else {
                            opp.team  = m.teamA
                            opp.score  = m.scoreA
                            home.team = m.teamB
                            home.score = m.scoreB
                        }
                    }
                    return (
                        <tr key={m.matchId} className="clickable-row" onClick={() => {
                            navigate(`/MatchPage?matchId=${m.matchId}`)
                        }}>
                            {/* <Link to={`/MatchPage?matchId=${m.matchId}`}></Link> */}
                            <td>{m.date || '—'}</td>
                            <td>{m.map}</td>
                            <td>{home.team} vs {opp.team}</td>
                            <td className={home.score > opp.score ? 'cell-kd-good' : 'cell-kd-bad'}>
                            {home.score} – {opp.score}
                            </td>
                            
                        </tr>
                    )})}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SeasonPage;