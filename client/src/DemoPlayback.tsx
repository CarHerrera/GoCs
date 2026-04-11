import { useEffect, useState, useRef } from "react";
import { Layer, Stage, Text, Circle, Group} from 'react-konva';
import { URLImage } from "./URLImage";
import Konva from "konva";
// interface Vector {
//     X: number;
//     Y: number;
//     Z: number;
// }
interface PlayerState {
    vector: MapCoordinate;
    weapon: string;
}
interface MapCoordinate  {
    X: number;
    Y: number;
}
interface MatchEvents {
    rounds: Record<number, RoundEvents>;
    map: {
        pos_x: string,
        pos_y: string,
        scale: string
    }
    teams: Record<string, Record<string, string>>;
}

interface RoundEvents {
    player_positions: Record<number, Record<string, PlayerState>>;
    player_info: Record<string, PlayerInformation>;
}

interface PlayerInformation {
    name: string;
    side: number
}
interface PlaybackState {
    playing: boolean
    round_no: number
    tick_no: number
}
function DemoPlayback({file, map}:{file:String, map:String}){
        const [stats, setStats] = useState<MatchEvents>()
        const [size, setSize] = useState({ width: 0, height: 0 });
        const [round, setRound] =useState(1);
        const [isPlaying, setPlaying] = useState<PlaybackState>({playing: false, round_no:1, tick_no: 0});
        const containerRef = useRef<HTMLDivElement>(null);
        const round_begin_ticks = useRef<Map<number,number>>(new Map<number,number>());
        const tickRef = useRef(0);
        const playerRef = useRef<Map<string, Konva.Group>>(null)
        const playbackRef = useRef<Map<number, (Map<number, Map<string, PlayerState>>)>>(new Map<number, (Map<number, Map<string, PlayerState>>)>);
        const layerRef = useRef<Konva.Layer>(null)
        // const lastWep = useRef("")
        function getMap(){
            if(!playerRef.current){
                playerRef.current = new Map();
            }
            return playerRef.current;
        }
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
                    return data;
                })
            }
            async function getFetch(){
                const data = await getStats();
                if(!ignore){
                    setStats(data)
                    // console.log(data)
                    round_begin_ticks.current = new Map<number,number>();
                }
            }
            getFetch()
            return () => { ignore = true}
        }, [file])
        // Handle Responsive Resizing
         useEffect(() => {
            const updateSize = () => {
                    if (containerRef.current) {
                        const parentElement = containerRef.current.parentElement;
                    if (!parentElement) return;

                    const availableHeight = parentElement.offsetHeight;
                    const availableWidth = parentElement.offsetWidth;

                    // Use the smaller dimension to keep it a square
                    const side = Math.min(availableHeight, availableWidth);                    
                    setSize({ width: side, height: side });
                }
        };

        const observer = new ResizeObserver(updateSize);
        if (containerRef.current) {
            observer.observe(containerRef.current);
        }

        updateSize(); 
        let elapsed = 0;
        const anim = new Konva.Animation((frame) => {
            if (!isPlaying.playing) {return}
            playerRef.current?.forEach((c, k) => {
                if (stats != null){
                    const oldround = stats?.rounds[round]
                    
                    const positions = oldround?.player_positions;
                    if (!positions) {
                        return;
                    }

                    if (tickRef.current === 0) {
                        tickRef.current = isPlaying.tick_no;
                    }
                    // tickRef.current += 1;
                    if(elapsed > 50){
                        tickRef.current += 1;
                        elapsed = 0
                    } else {
                        elapsed += Math.round(frame.timeDiff)
                    }
                    console.log(`TICK:${tickRef.current}`)
                    if (playbackRef.current.get(round)?.has(tickRef.current)) {
                        const positions = playbackRef.current.get(round)!.get(tickRef.current)?.get(k)
                        c.getChildren().forEach((g) => {
                            // Positions become null means that the player died in the server because we stopped recording them
                            // TODO Maybe change it to an X if I can?
                            if (g.className == "Circle") {
                                if(positions == null){
                                    g.x(-1000);
                                    g.y(-1000)
                                } else {
                                    g.x(positions!.vector.X);
                                    g.y(positions!.vector.Y)
                                }
                                
                            } else {
                                if(positions == null){
                                    g.x(-1000);
                                    g.y(-1000)
                                } else {
                                    g.x(positions!.vector.X+5);
                                    g.y(positions!.vector.Y-3)
                                }
                                
                            }
                        })
                    }                        
                }
                
            })
        }, layerRef.current);
        anim.start()
        
        return () => {
            observer.disconnect()
            anim.stop()
        };
        }, [isPlaying.playing, round]);  

        let playerecords : [string, string, number, number, number][]  = []
        if (stats != null){
            
            const rounds = Array.from(Object.entries(stats.rounds))
            const {pos_x, pos_y, scale} = stats.map
            const originX = parseFloat(pos_x);
            const originY = parseFloat(pos_y);
            const mapScale = parseFloat(scale);
            const newX = (x:number) => {
                return (x-originX)/mapScale * size.width/1024
            }
            const newY = (y:number) => {
                return (originY-y)/mapScale * size.height/1024
            }
            rounds.forEach(([round_no, round_ev]) => {
                const pos = Array.from(Object.entries(round_ev.player_positions))
                const tick_map = new Map<number, Map<string, PlayerState>>();
                pos.forEach(([tick, playervec],i) => {
                    const info = Array.from(Object.entries(playervec))
                    const player_pos = new Map<string, PlayerState>()
                    info.forEach(([playerid, state]) =>{
                        const place:MapCoordinate = {
                            X:newX(state.vector.X), Y:newY(state.vector.Y)
                        }
                        const player_state: PlayerState = {
                            vector: place, weapon: state.weapon
                        }
                        player_pos.set(playerid, player_state)
                    })
                    tick_map.set(Number(tick), player_pos)
                    if (i ==0) {
                        tick_map.set(0, player_pos)
                        round_begin_ticks.current.set(Number(round_no), Number(tick))
                    }
                });
                playbackRef.current.set(Number(round_no), tick_map)
            })
            const player_info = Array.from(Object.entries(rounds[round-1][1].player_info))
            player_info.forEach(([playerid, playername]) => {
                const playerpos = playbackRef.current.get(round)!.get(0)?.get(playerid)
                playerecords.push([playerid, playername.name, playerpos!.vector.X, playerpos!.vector.Y, playername.side ])
            })
            
        }
        console.log(stats)
    return <>
        <div className="playbackGrid" >
            {stats && Array.from(Object.entries(stats!.teams)).map(([teamname, players],i) => {
                const player_names = Array.from(Object.values(players));
                return (<>
                        <div className={`team${i+1}`} key={i}>
                            <h3>{teamname}</h3>

                            {player_names.map((p,j) => {
                                return <div key={j}>{p}</div>
                            })}
                        </div>
                </>)
            })}
            {
                stats == null &&
                <>
                    <div className="team1">
                        Team 1
                        Current Round: {round}
                    </div>
                    <div className="team2">
                        Team 2
                    </div>
                </>
            }
            <div className="playbackMap" ref={containerRef} >
                <Stage width={size.width} height={size.height}>
                    <Layer ref={layerRef}>
                       <URLImage src={`/overviews/${map}.jpg`}  width={size.width} height={size.height}></URLImage>       
                        { stats && 
                            Array.from(Object.entries(stats!.rounds[round].player_info)).map(([playerid, playerinfo],i) => {
                                const color = playerinfo.side == 2 ? "orange" : "blue"
                                const pos = playbackRef.current!.get(round)!.get(0)!.get(playerid)
                                return (
                                    <Group key={i} ref={(node) =>{
                                                const map = getMap();
                                                if (node != null) {
                                                    map.set(playerid, node)
                                                }
                                                return () => {map.delete(playerid)}
                                            }}>
                                        <Circle
                                            x={pos!.vector.X}
                                            y={pos!.vector.Y}
                                            fill={color}
                                            radius={5}
                                        />
                                        <Text 
                                            text={playerinfo.name} 
                                            x={pos!.vector.X + 5} 
                                            y={pos!.vector.Y - 3} 
                                            fill="white" 
                                            fontSize={10} 
                                        />
                                    </Group>
                                );        
                            })
                        } 
                        
                    </Layer>
                </Stage>
            </div>
            
            <div className="player">
                <button>Something</button>
                <button onClick={() => setPlaying({...isPlaying, playing: !isPlaying.playing})}>{ isPlaying.playing == false ? "Play": "Pause"}</button>
            </div>
            <div className="progress">
                Progtess Bar
            </div>
            <div className="rounds">
                <ul style={{justifyContent:"center", marginTop:"10px"}}>
                {stats && 
                    Array.from(Object.keys(stats.rounds)).map((s, i) => {
                        return <li key={i} onClick={()=>{
                            const tickNo = round_begin_ticks.current.get(i + 1);
                            if (tickNo === undefined) {
                                return;
                            }
                            tickRef.current = tickNo
                            setRound(i+1); setPlaying({...isPlaying, playing: false, 
                                tick_no:tickNo, round_no:i+1
                        })}}>{s}</li>
                    })
                }
                </ul>
            </div>
        </div>
    </>
}

export default DemoPlayback;  