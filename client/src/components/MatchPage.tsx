import { useSearchParams } from "react-router-dom";
import { useState, useEffect } from 'react';
import '../css/stats.css'
import DemoPlayback from "./DemoPlayback";
import AdvancedStatsPanel from "./AdvancedStatsSubTab";
import UnassignedSection from "./UnassignedTab";
import type {Stats} from "../helpers/types"


const TABS = [
    { id: 1, label: "Overview" },
    { id: 2, label: "Advanced Stats" },
    // { id: 3, label: "HeatMaps" },
    { id: 4, label: "2D Playback" },
] as const;

function NavBar({activeTab, setActive}:{activeTab: number, setActive: React.Dispatch<React.SetStateAction<number>>}){
    return (
        <div className="subtabs">
            {TABS.map(tab => (
                <span
                    key={tab.id}
                    onClick={() => setActive(tab.id)}
                    className={activeTab === tab.id ? "active" : ""}
                >
                    {tab.label}
                </span>
            ))}
        </div>
    );
}

function TeamTable({team}: {team: Stats}){
    const players = Object.entries(team.Playing ?? {})
        .sort(([, a], [, b]) => b.stats.kills - a.stats.kills);

    return (
        <div>
            <h3>{team.Clanname}</h3>
            <table className="statsTable">
                <thead>
                    <tr>
                        <td>Player</td>
                        <td>K</td>
                        <td>A</td>
                        <td>D</td>
                        <td>KD Ratio</td>
                        <td>2K</td>
                        <td>3K</td>
                        <td>4K</td>
                        <td>5K</td>
                        <td>ADR</td>
                        <td>UD</td>
                        <td>KAST%</td>
                        <td>HS%</td>
                        <td>Rating</td>
                    </tr>
                </thead>
                <tbody>
                    {players.map(([id, player]) => {
                        const ratio = player.stats.kills/ player.stats.deaths;
                        const hsPct = player.stats.kills > 0
                            ? (player.stats.hs / player.stats.kills) * 100
                            : 0;
                        return (
                            <tr key={id}>
                                <td>{player.name}</td>
                                <td>{player.stats.kills}</td>
                                <td>{player.stats.assists}</td>
                                <td>{player.stats.deaths}</td>
                                <td className={ratio > 1 ? "pos" : ratio  < 1 ? "neg" : ""}>
                                    {ratio.toFixed(2)}
                                </td>
                                <td>{player.stats["2k"]}</td>
                                <td>{player.stats["3k"]}</td>
                                <td>{player.stats["4k"]}</td>
                                <td>{player.stats["5k"]}</td>
                                <td>{player.stats.adr.toFixed(1)}</td>
                                <td>{player.stats.ud}</td>
                                <td>{player.stats.kast.toFixed(1)}%</td>
                                <td>{hsPct.toFixed(0)}%</td>
                                <td className="rating-cell">{player.stats.rating.toFixed(2)}</td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}

function Overview ({stats}:{stats: Stats[]}){
    if (!stats || stats.length < 2) {
        return <div className="loading">Loading match stats...</div>;
    }
    
    return (
        <div className="stats">
            <TeamTable team={stats[0]} />
            <TeamTable team={stats[1]} />
        </div>
    );
}

function SideBreakdown({team}: {team: Stats}){
    return (
        <span className="side-breakdown">
            <span className="sb-dot t" /> {team.TScore}
            <span className="sb-dot c" /> {team.CTScore}
        </span>
    );
}

function MatchPage(){
    const [searchParams] = useSearchParams();
    const [stats, setStats] = useState<Stats[]>([])
    const [map, setMap] = useState<string>("");
    const [activeTab, setActive] = useState<number>(1)
    const matchId = searchParams.get("matchId")
    useEffect(() => {
        fetch(`http://localhost:4000/api/match/advancedStats/${matchId}`, { credentials: 'include' })
          .then(r => r.json())
          .then((data: { mapName: string; stats: Stats[] }) => { 
            console.log(data)
            setStats(data.stats);
            setMap(data.mapName);
           })
          .catch(() => {})
      }, [matchId]);

    let frontPage = (<>Hello</>);
    if (activeTab == 1){
        frontPage = (<Overview stats={stats}></Overview>)
    } else if (activeTab == 2) {
        frontPage = (<AdvancedStatsPanel TeamA={stats[0]} TeamB={stats[1]} matchId={Number(matchId)}></AdvancedStatsPanel>)
    }else if (activeTab == 4){
        frontPage = (<DemoPlayback matchId={Number(matchId) ?? -1} map={map}></DemoPlayback>)
    }

    return (<>
        <div>
            <div className="header match-header">
                <h1>
                    {stats && stats.length !=0 ? stats[0].Clanname : "Team 1"}
                    <span className="vs"> vs </span>
                    {stats && stats.length !=0 ? stats[1].Clanname : "Team 2"}
                </h1>
                {stats && stats.length != 0 && (
                    <div className="score-row">
                        <span className="score-main">{stats[0].Endscore} — {stats[1].Endscore}</span>
                        <span className="score-detail">
                            <SideBreakdown team={stats[0]} /> &nbsp;·&nbsp; <SideBreakdown team={stats[1]} />
                        </span>
                    </div>
                )}
                <NavBar activeTab={activeTab} setActive={setActive}></NavBar>
            </div>

            {matchId && Number(matchId) > 0 && (
                <UnassignedSection matchId={Number(matchId)} hideWhenEmpty />
            )}

            {activeTab == 1 ? (
                <div className="grid">
                    <div className="map">
                        <img style={{display:"inline"}} src={`/overviews/${map}.jpg`} alt={`${map} overview`}></img>
                    </div>
                    {frontPage}
                </div>
            ) : (frontPage)
            }
        </div>
    </>)
}

export default MatchPage;