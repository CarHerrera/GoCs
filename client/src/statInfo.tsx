import { useSearchParams } from "react-router-dom";
import { useState, useEffect } from 'react';
import './css/stats.css'
import DemoPlayback from "./DemoPlayback";
interface Stats{
    ID: number,
    Clanname: string,
    Endscore: number,
    Tscore: number,
    CTScore: number,
    Playing: Record<string, Player>
}
interface Player{
    name: string,
    ID: number,
    stats: PlayerStats
}
interface PlayerStats{
    kills: number,
    deaths: number,
    assists: number
}
interface Match {
    Rounds: Record<number, Rounds>,
}
interface Rounds{
    InGamePlayers: Record<string, Positions>
}
interface Positions {
    Positions:  []
}
function NavBar({setActive}:{setActive :React.Dispatch<React.SetStateAction<number>>}){
    return <>
        <ul>
            <li onClick={() => setActive(1)}>Overview</li>
            <li onClick={() => setActive(2)}>Advanced Stats</li>
            <li onClick={() => setActive(3)}>HeatMaps</li>
            <li onClick={() => setActive(4)}>2D Playback</li>
        </ul>
    </>
}
function Playback({search}:{search:URLSearchParams}) {
    // const map = search.get("map");
    const [stats, setStats] = useState<Match>()
    const file = search.get("file");
    useEffect(( ) => {
        let ignore = false;
        async function getStats(){
            return fetch(`http://localhost:4000/2DPlayback/${file}`, {
                method: "GET",
                headers: {
                    accept:"Application/JSON"
                }
            })
            .then(response => {
                if(!response.ok){
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            }) .then(data => {
                // console.log(data)
                return data;
            })
        }
        async function getFetch(){
            const data = await getStats();
            if(!ignore){
                setStats(data)
                console.log(data)
                // console.log(stats)
            }
        }
        getFetch()
        return () => { ignore = true}
    }, [file])
    return <><DemoPlayback></DemoPlayback></>
}
function Overview ({stats}:{stats: Stats[]}){
    return (
        <div className="stats">
                <div>
                    <h3>{stats.length !=0 ? stats[0].Clanname : ""}</h3>
                    <table className="statsTable">
                        <thead>
                            <tr>
                            <td>Player</td>
                            <td>Kills</td>
                            <td>Assists</td>
                            <td>Deaths</td>
                            </tr>
                        </thead>
                        <tbody>
                            {   stats.length !=0 ?
                                Object.entries(stats[0].Playing).map(([name, player], i) => {
                                    return <tr key ={i}><td>{name}</td><td>{player.stats.kills}</td><td>{player.stats.assists}</td><td>{player.stats.deaths}</td></tr>
                                }) : ""
                            }
                        </tbody>
                    </table>
                </div>
                <div>
                    <h3>{stats.length !=0 ? stats[1].Clanname : ""}</h3>
                    <table className="statsTable">
                        <thead>
                            <tr>
                            <td>Player</td>
                            <td>Kills</td>
                            <td>Assists</td>
                            <td>Deaths</td>
                            </tr>
                        </thead>
                        <tbody>
                            {   stats.length !=0 ?
                                Object.entries(stats[1].Playing).map(([name, player], i) => {
                                    return <tr key ={i}><td>{name}</td><td>{player.stats.kills}</td><td>{player.stats.assists}</td><td>{player.stats.deaths}</td></tr>
                                }): ""
                            }
                        </tbody>
                    </table>
                </div>
            </div>
    )
}
function AdvancedStats(){
    const [searchParams] = useSearchParams();
    const [stats, setStats] = useState<Stats[]>([])
    const [activeTab, setActive] = useState<number>(1)
    const map = searchParams.get("map");
    const file = searchParams.get("file");

    useEffect(() => {
        let ignore = false;
        async function getStats(){
            return fetch(`http://localhost:4000/advancedStats/${file}`, {
                method: "GET",
                headers: {
                    accept:"Application/JSON"
                }
            })
            .then(response => {
                if(!response.ok){
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            }) .then(data => {
                return data;
            })
        }
        async function getFetch(){
            const data = await getStats();
            if(!ignore){
                setStats(data)
                console.log(data)
            }
        }
        getFetch()
        return () => { ignore = true}
    }, [file])

    let frontPage = (<>Hello</>);
    if (activeTab == 1){
        frontPage = (<Overview stats={stats}></Overview>)
    } else if (activeTab == 4){
        frontPage = (<Playback search={searchParams}></Playback>)
    }

    // console.log(searchParams)
    // Add Some Check to see if the demo has been parsed before via a SQL check
    return (<>
        <div className="grid">
            <div className="header">
                <h1>{stats.length !=0 ? stats[0].Clanname : "Team 1"} vs {stats.length !=0 ? stats[1].Clanname : "Team 2"}</h1>
                {stats.length != 0 ? `${stats[0].Endscore} - ${stats[1].Endscore}`: ""}
                <NavBar setActive={setActive}></NavBar>
            </div>
            <div className="map">
                <img style={{display:"inline"}}src={`/src/assets/overviews/${map}.jpg`}></img>
            </div>
            {frontPage}
        </div>
        
    </>)
}

export default AdvancedStats;  