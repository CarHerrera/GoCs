import { useSearchParams } from "react-router-dom";
import React, { useState, useEffect } from 'react';
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
function AdvancedStats(){
    const [searchParams] = useSearchParams();
    const [stats, setStats] = useState<Stats[]>([])
    const map = searchParams.get("map");
    const file = searchParams.get("file");
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
    useEffect(() => {
        let ignore = false;
        async function getFetch(){
            const data = await getStats();
            if(!ignore){
                setStats(data)
            }
        }
        getFetch()
        return () => { ignore = true}
    }, [])
    let team1 : Stats;
    let team2 : Stats;
    if (stats.length !=0){
         team1  = stats[0]
         team2 = stats[1]

    } else {
         team1 =  team2 ={
                ID: -1,
                Clanname: "NULL",
                Endscore: -1,
                Tscore: -1,
                CTScore: -1,
                Playing: {},
            
         };    
    }  
    
    // console.log(searchParams)
    // Add Some Check to see if the demo has been parsed before via a SQL check
    return (<>
        Hello World {map}! 
        {file}
        <div style={{display:"flex"}}>
            <div>
                <h1>{stats.length !=0 ? stats[0].Clanname : ""}</h1>
                <table>
                    <thead>
                        <tr>
                        <td>Player</td>
                        <td>Kills</td>
                        <td>Assists</td>
                        <td>Deaths</td>
                        </tr>
                    </thead>
                    <tbody>
                        {
                            Object.entries(team1.Playing).map(([name, player], i) => {
                                return <tr key ={i}><td>{name}</td><td>{player.stats.kills}</td><td>{player.stats.assists}</td><td>{player.stats.deaths}</td></tr>
                            })
                        }
                    </tbody>
                </table>
            </div>
            <div style={{flexGrow:"1"}}><img src={`/src/assets/overviews/${map}.jpg`}></img></div>
            <div>
                <h1>{stats.length !=0 ? stats[1].Clanname : ""}</h1>
                <table>
                    <thead>
                        <tr>
                        <td>Kills</td>
                        <td>Assists</td>
                        <td>Deaths</td>
                        </tr>
                    </thead>
                    <tbody>
                        {
                            Object.entries(team2.Playing).map(([name, player], i) => {
                                return <tr key ={i}><td>{name}</td><td>{player.stats.kills}</td><td>{player.stats.assists}</td><td>{player.stats.deaths}</td></tr>
                            })
                        }
                    </tbody>
                </table>
            </div>
        </div>
        
    </>)
}

export default AdvancedStats;  