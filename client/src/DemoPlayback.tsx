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
interface GrenadeState{
    vector: MapCoordinate;
    grenade: string;
    thrownBy: string;
    thrownById: string;
    status: string;
}
interface FireState{
    vertices: MapCoordinate[];
    status: string;
}
interface MapCoordinate  {
    X: number;
    Y: number;
}
interface MatchEvents {
    round_events: RoundEvents;
    rounds: number;
    map: {
        pos_x: string,
        pos_y: string,
        scale: string
    }
    teams: Record<string, Record<string, string>>;
}

interface RoundEvents {
    // Ticks -> STEAMID -> PLAYER STATE
    player_positions: Record<number, Record<string, PlayerState>>;
    // PLAYERID -> NAME
    player_info: Record<string, PlayerInformation>;
    // TICKS -> UTILID -> GRENADE STATE
    grenade_events: Record<number, Record<string, GrenadeState>>
    // TICKS -> ENTID -> FIRE STATE
    fire_events: Record<number, Record<string, FireState>>
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
interface PlayBackRef{
    // TICK NO -> ROUND_PLAYBACK
    player_pos: Map<number, Map<string, PlayerState>>;
    grenade_pos: Map<number, Map<string, GrenadeState>>;
    fire_vertices: Map<number, Map<string, FireState>>;
}
function DemoPlayback({file, map}:{file:String, map:String}){
        const [stats, setStats] = useState<MatchEvents>()
        const [size, setSize] = useState({ width: 0, height: 0 });
        const [round, setRound] =useState(1);
        const [isPlaying, setPlaying] = useState<PlaybackState>({playing: false, round_no:1, tick_no: 0});
        const containerRef = useRef<HTMLDivElement>(null);
        const round_begin_ticks = useRef<number[]>([]);
        const tickRef = useRef(0);
        const playerRef = useRef<Map<string, Konva.Group>>(null);
        // ROUND NO -> TICK NO -> PLAYBACK REF
        const playbackRef = useRef<PlayBackRef>(null);
        const layerRef = useRef<Konva.Layer>(null)
        // const lastWep = useRef("")
        function getPlayerRef(){
            if(!playerRef.current){
                playerRef.current = new Map();
            }
            return playerRef.current;
        }
        useEffect(( ) => {
            let ignore = false;
            async function getStats(){
                return fetch(`http://localhost:4000/2DPlayback/${file}-${round}`, {
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
                }
            }
            getFetch()
            return () => { ignore = true}
        }, [file, round])
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
        const anim = new Konva.Animation((frame) => {
            if (!isPlaying.playing) {
                tickRef.current = isPlaying.tick_no
                return
            }

            if (playerRef.current != null){
                if (tickRef.current === 0) {
                    tickRef.current = isPlaying.tick_no;
                }
               tickRef.current += 1;
                playerRef.current?.forEach((group_object, itemid) => {
                    const x = group_object!.getAttr("name")
                        if (x == "player"){    
                            if (playbackRef.current!.player_pos.has(tickRef.current)){
                                const positions = playbackRef.current!.player_pos.get(tickRef.current)!.get(itemid)
                                group_object.getChildren().forEach((g) => {
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
                                    
                                }})
                            }
                        } else {
                            // found a group that isn't a grenade'
                            if (group_object.hasName("FLYING")){
                                if(playbackRef.current!.grenade_pos.has(tickRef.current)){
                                    const gren_pos = playbackRef.current!.grenade_pos.get(tickRef.current)!.get(itemid)
                                    if (gren_pos != null && gren_pos.status != "FLYING"){
                                        if (gren_pos.status == "BLOOMED") {
                                            group_object.removeName("FLYING")
                                            group_object.addName("BLOOMED")
                                            group_object.getChildren().forEach((g) => {
                                                if (g.className == "Circle") {
                                                    const circle = g as Konva.Circle
                                                    circle.radius(size.height * .035);
                                                }
                                            })
                                            console.log(`CHANGED AT ${tickRef.current} ${gren_pos.grenade} ${gren_pos.status}`)
                                        }

                                        if (gren_pos.status == "EXPIRED" || gren_pos.status == "LANDED"){
                                            group_object.destroyChildren()              
                                            playerRef.current?.delete(itemid)
                                            getPlayerRef().delete(itemid)
                                            console.log(`DELETED AT ${tickRef.current} ${gren_pos.grenade} ${gren_pos.status}`)
                                        }
                                        
                                        
                                        
                                    }
                                    group_object.getChildren().forEach((g) => {
                                        if (g.className == "Circle") {
                                            if(gren_pos != null){
                                                g.x(gren_pos!.vector.X);
                                                g.y(gren_pos!.vector.Y)  
                                            } 
                                        } else {
                                            if(gren_pos != null){
                                                g.x(gren_pos!.vector.X+5);
                                                g.y(gren_pos!.vector.Y-3)
                                            }
                                            
                                        }
                                    })
                                }
                            } else if (group_object.hasName("BLOOMED")){
                                if(playbackRef.current!.grenade_pos.has(tickRef.current)){
                                    const gren_pos = playbackRef.current!.grenade_pos.get(tickRef.current)!.get(itemid)
                                    if (gren_pos != null && gren_pos.status == "EXPIRED"){
                                        group_object.destroyChildren()
                                        playerRef.current!.delete(itemid)
                                        getPlayerRef().delete(itemid)
                                        console.log(`DELETED AT ${tickRef.current} ${gren_pos.grenade} ${gren_pos.status}`)
                                    }
                                }
                            }
                            // FIRE
                            if(group_object.hasName("STARTING")){
                                if (playbackRef.current!.fire_vertices.has(tickRef.current)){
                                    const fire_info = playbackRef.current!.fire_vertices.get(tickRef.current)!.get(itemid)
                                    if (fire_info != null){
                                        // Fire starts as circle since only one vertex.
                                        //  If more than one vertex and is still a circle then delete. Draw new shape
                                        if (fire_info.vertices.length > 1){
                                            group_object.destroyChildren()
                                            group_object.removeName("STARTING")
                                            group_object.addName(fire_info.status)

                                            const shape:Konva.Shape = new Konva.Shape({
                                                fill:"orange",
                                                sceneFunc: function (context,shape) {
                                                    fire_info.vertices.forEach((vertex, i) => {
                                                            if (i == 0){
                                                                context.moveTo(vertex.X, vertex.Y)
                                                                context.beginPath()
                                                            } else {
                                                                context.lineTo(vertex.X, vertex.Y)
                                                            }
                                                    })
                                                    context.closePath()
                                                    context.fillStrokeShape(shape)
                                                }
                                            })
                                            group_object.add(shape)
                                            group_object.addName(`${fire_info.vertices.length}`)
                                        }
                                        
                                    }
                                }
                            } else if (group_object.hasName("SPREADING")){
                                if (playbackRef.current!.fire_vertices.has(tickRef.current)){
                                    const fire_info = playbackRef.current!.fire_vertices.get(tickRef.current)!.get(itemid)
                                    if (fire_info != null){
                                        const curr_vertices = parseInt(group_object.name()[group_object.name().length-1])
                                        if (fire_info.status == "ENDING"){
                                            group_object.destroyChildren()
                                            playerRef.current?.delete(itemid)
                                            getPlayerRef().delete(itemid)
                                        } else {
                                            if (curr_vertices != fire_info.vertices.length){
                                                group_object.destroyChildren()
                                                group_object.removeName(`${curr_vertices}`)
                                                const shape:Konva.Shape = new Konva.Shape({
                                                    fill:"orange",
                                                    sceneFunc: function (context,shape) {
                                                        fire_info.vertices.forEach((vertex, i) => {
                                                                if (i == 0){
                                                                    context.moveTo(vertex.X, vertex.Y)
                                                                    context.beginPath()
                                                                } else {
                                                                    context.lineTo(vertex.X, vertex.Y)
                                                                }
                                                        })
                                                        context.closePath()
                                                        context.fillStrokeShape(shape)
                                                    }
                                                })
                                                group_object.add(shape)
                                                group_object.addName(`${fire_info.vertices.length}`)
                                            }
                                        }
                                    }
                                }
                            }
                            
                        }
                        // Check if a grenade needs to be created   
                })
                if (playbackRef.current!.grenade_pos.has(tickRef.current)) {
                    const map = getPlayerRef()
                    const tickMap = playbackRef.current!.grenade_pos.get(tickRef.current)
                    tickMap?.forEach((state, id) => {
                        if (map.has(id)){
                            return;
                        } else {
                            if (state.status == "EXPIRED" || state.status == "LANDED" || state.status == "ENDING") { return }                    
                            let gren = new Konva.Group({name:`${state.grenade} ${state.status}`, id:id})
                            let circl = new Konva.Circle({
                                    x: state.vector.X, y: state.vector.Y, radius: size.width * .01 , fill:"white"
                                })
                            let label = new Konva.Text({
                                    x: state.vector.X + 5, y: state.vector.Y - 3, text: state.grenade,
                                    fill:"white" , fontSize:10 
                            })
                            gren.add(circl, label)
                            console.log(`CREATED AT ${tickRef.current} ${state.grenade} ${state.status}`)
                            console.log(gren)
                            map.set(id, gren)
                            layerRef.current!.add(gren);
                        }
                    })
                }

                if(playbackRef.current!.fire_vertices.has(tickRef.current)){
                    const map = getPlayerRef()
                    const tickMap = playbackRef.current!.fire_vertices.get(tickRef.current)
                    tickMap?.forEach((state, id) => {
                        if (map.has(id)){
                            return;
                        } else {
                            if (state.status == "ENDING") {return}
                            let fire = new Konva.Group({name:`fire ${state.status}`})
                            let circl = new Konva.Circle({
                                x : state.vertices[0].X , y: state.vertices[0].Y, radius: size.height * .02, fill:"orange"
                            })
                            fire.add(circl)
                            map.set(id, fire)
                            layerRef.current!.add(fire)
                        }
                    })
                }
            }
        }, layerRef.current);
        anim.start()       
        return () => {
            observer.disconnect()
            anim.stop()
        };
        }, [isPlaying.playing, round]);  
        let playerecords : [string, string, number, number, number][]  = []
        if (stats != null){
            console.log(stats)
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
            const pos = Array.from(Object.entries(stats.round_events.player_positions))
            const grenades = Array.from(Object.entries(stats.round_events.grenade_events))
            const fire_vertices = Array.from(Object.entries(stats.round_events.fire_events))
            let tick_map:PlayBackRef = {
                player_pos: new Map<number, Map<string, PlayerState>>(),
                grenade_pos: new Map<number, Map<string, GrenadeState>>(),
                fire_vertices: new Map<number, Map<string, FireState>>(),
            };
            // Assume that each tick in PlayerState also exists in Grenade
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
                
                tick_map.player_pos.set(Number(tick), player_pos)
                if (i ==0) {
                    tick_map.player_pos.set(0, player_pos)
                    round_begin_ticks.current.push(Number(tick))
                    tickRef.current = Number(tick)
                }
                if ((info.length-1) == i){
                    round_begin_ticks.current.push(Number(tick))
                }
            });
            grenades.forEach(([tick, grenadeEvent]) => {
                const grenade_info = Array.from(Object.entries(grenadeEvent))
                const grenade_pos = new Map<string, GrenadeState>();
                grenade_info.forEach(([grenid, grenstate],i) => {
                    const place:MapCoordinate = {
                        X:newX(grenstate.vector.X), Y:newY(grenstate.vector.Y)
                    }
                    const grenade_state: GrenadeState = {
                        vector: place, thrownBy: grenstate.thrownBy, thrownById: grenstate.thrownById, grenade:grenstate.grenade, status:grenstate.status
                    }
                    grenade_pos.set(grenid, grenade_state)
                })
                tick_map.grenade_pos.set(Number(tick), grenade_pos)
            })
            fire_vertices.forEach(([tick, fire_event]) => {
                const fire_info = Array.from(Object.entries(fire_event))
                const fire = new Map<string, FireState>();

                fire_info.forEach(([entid, state]) => {
                    const vertices:MapCoordinate[] = [];
                    state.vertices.forEach((pos) => {
                        const place:MapCoordinate = {
                            X: newX(pos.X), Y:newY(pos.Y)
                        }
                        vertices.push(place)
                    })
                    const fire_state:FireState = {
                        vertices: vertices, status: state.status
                    }
                    fire.set(entid, fire_state)
                })
                tick_map.fire_vertices.set(Number(tick), fire)
            })
            playbackRef.current = tick_map
            // console.log(playbackRef.current)
            const player_info = Array.from(Object.entries(stats.round_events.player_info))
            player_info.forEach(([playerid, playername]) => {
                const playerpos = playbackRef.current!.player_pos.get(0)?.get(playerid)
                playerecords.push([playerid, playername.name, playerpos!.vector.X, playerpos!.vector.Y, playername.side ])
            })
            
        }
        
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
                            Array.from(Object.entries(stats!.round_events.player_info)).map(([playerid, playerinfo],i) => {
                                const color = playerinfo.side == 2 ? "orange" : "blue"
                                const pos = playbackRef.current!.player_pos.get(0)!.get(playerid)
                                return (
                                    <Group key={i} name={"player"} ref={(node) =>{
                                                const map = getPlayerRef();
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
                <button onClick={() => {
                    tickRef.current -= 500
                    console.log(tickRef.current)
                    // This is to reset the icons and clear grenades until a better solution is found.
                    playerRef.current?.forEach((group, key) => {
                        const groupobject = group.getAttr("name")
                        if (groupobject == "player"){
                            return
                        } else {
                            group.destroyChildren()
                            
                            playerRef.current?.delete(key)
                            getPlayerRef().delete(key)
                        }
                    })
                    }}>Back</button>
                <button onClick={() => {
                        console.log(`TICK:${tickRef.current}`)
                        setPlaying({...isPlaying, tick_no:tickRef.current, playing: !isPlaying.playing}) 
                    }}>{ isPlaying.playing == false ? "Play": "Pause"}</button>
                <button onClick={() => {
                    tickRef.current += 500
                    console.log(tickRef.current)
                    playerRef.current?.forEach((group, key) => {
                        const groupobject = group.getAttr("name")
                        if (groupobject == "player"){
                            return
                        } else {
                            group.destroyChildren()
                            
                            playerRef.current?.delete(key)
                            getPlayerRef().delete(key)
                        }
                    })
                    }}>forward</button>
            </div>
            <div className="progress">
                Progtess Bar
            </div>
            <div className="rounds">
                <ul style={{justifyContent:"center", marginTop:"10px"}}>
                {stats && 
                        Array.from({length:stats.rounds}, (_, i) => i+1).map((n, j) => {
                            return <li key={j} onClick={() => {
                                setRound(n);
                                playerRef.current?.forEach((group, key) => {
                                    const groupobject = group.getAttr("name")
                                    if (groupobject == "player"){
                                        return
                                    } else {
                                        group.destroyChildren()
                                        
                                        playerRef.current?.delete(key)
                                        getPlayerRef().delete(key)
                                    }
                                })
                                setPlaying({tick_no:round_begin_ticks.current[1], playing:false, round_no: n})
                            }}className={(n) == isPlaying.round_no ? "activeRound" : ""}>
                                {n}
                            </li>
                        })
                }
                </ul>
            </div>
        </div>
    </>
}

export default DemoPlayback;  