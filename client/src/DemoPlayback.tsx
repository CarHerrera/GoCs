import { useEffect, useState, useRef } from "react";
import { Layer, Stage, Text, Circle, Group, Rect, Image} from 'react-konva';
import { URLImage } from "./URLImage";
import Konva from "konva";
import useImage from "use-image";
import { type_to_svg } from './helpers/equipIdToSvg'
const PlayerAction = {
    isMoving: 1,
    beginPlanting: 2,
    donePlanting: 3,
    abortedPlant: 4,
} as const;
type PlayerAction = typeof PlayerAction[keyof typeof PlayerAction];
interface PlayerState {
    vector: MapCoordinate;
    active_weapon: number;
    hp: number;
    kills: number;
    assists: number;
    deaths: number;
    armor: number;
    dinero: number;
    action: PlayerAction;
    hasBomb: boolean;
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
interface PlayerInformation {
    name: string;
    side: number
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
    player_info: Map<string, PlayerInformation>;
}

interface EntityInfo{
    kind: string
    status: string
    id: string
    lastTickUpdate: number
}
interface EntityLife{
    grenadeString: string
    lastSavedState: string
    flyingStart: number
    bloomedStart?: number
    bloomExpiry?: number
    flyingExpire: number
}
interface PlayerBox {
    width: number
    height: number
    tpad: number
    lpad: number
    i: number
    name: string
    playerid: string
    weapon: number
    hp: number
    hasBomb: boolean
    money: number
    bomb: HTMLImageElement | undefined
}
interface PlayerHudCache {
    hpBar: Konva.Rect;
    hpText: Konva.Text;
    statsText: Konva.Text;
    activeWep: Konva.Image;
    bombImage: Konva.Image | null;
}
interface PlayerMapCache {
    circle: Konva.Circle
    Name: Konva.Text
}
type WeaponIconProps = Omit<Konva.ImageConfig, 'image'> & { src: string }
function WeaponIcon({src, ...rest}: WeaponIconProps){
    const [img] = useImage(src, 'anonymous')
    return img ? <Image {...rest} image={img}/> : null
}
function playerBoxInfo({playerBox, hud, cache}:{playerBox:PlayerBox, hud: Map<string, Konva.Group>, cache:Map<string, PlayerHudCache>}){
    let color ="white"
    const {width, height, tpad, lpad, i, name, playerid, weapon} = playerBox
    switch (i){
        case 0:
            color="green"
        break;
        case 1:
            color="blue"
        break;
        case 2:
            color="purple"
        break;
        case 3:
            color="black"
        break;
        case 4:
            color="orange"
        break;
        
    }
    const elements: Partial<PlayerHudCache> = {};
    const active_weapon = type_to_svg(weapon)
    if (i == 0) {
        const bottom = height - 20 - tpad/2;
        return <Group key={i} name={playerid} ref={(node) => {
                if (node != null){
                    hud.set(playerid, node)
                    cache.set(playerid, elements as PlayerHudCache)
                } else {
                    cache.delete(playerid);
                }
            }}>
                <Rect width={width} height={height} fill={color} stroke={"black"} strokeWidth={5}/>
                <Rect width={width} ref={(n) => {if (n) elements.hpBar = n}}  name="hpbar" height={height *.4} fill={"red"} stroke={"black"} strokeWidth={5}/>
                {(playerBox.hasBomb && playerBox.bomb) && (<Image image={playerBox.bomb} ref={(n) => {if (n) elements.bombImage = n}} name="bomb"  x={width - (width/10) - lpad/2}  y={bottom -tpad/2   } height={25} width={25} />)} 
                <Text name="hp" ref={(n) => {if (n) elements.hpText = n}} text={`${playerBox.hp}`} fill={"white"} x={width - lpad * 3}  y={tpad} fontSize={12} ></Text>
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${playerBox.money}`} fill={"white"} x={lpad}  y={bottom/2 + tpad} fontSize={12} ></Text>
                {/* <Text width={width} name="inv" text={`${weapon} HE,F,F,M 30/4`} fill={"white"} x={lpad}  y={bottom} fontSize={12} ></Text> */}
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}} src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
            </Group>
        
    } else {
        // This is where the rect starts
        const yStart = i * height + i *tpad;
        const bottom = yStart + height - 20 - tpad/2;
        const mid = yStart + (bottom - yStart)/2 + tpad
        return <Group key={i} name={playerid} ref={(node) => {
                if (node != null){
                    hud.set(playerid, node)
                    cache.set(playerid, elements as PlayerHudCache)
                } else {
                    cache.delete(playerid);
                }
            }}>
                <Rect width={width} height={height} fill={color} stroke={"black"} y={yStart}strokeWidth={5}/>
                <Rect width={width} ref={(n) => {if (n) elements.hpBar = n}} name="hpbar" height={height *.4} fill={"red"} stroke={"black"} y={yStart} strokeWidth={5}/>
                <Text name="hp" ref={(n) => {if (n) elements.hpText = n}}  text={`${playerBox.hp}`} fill={"white"} x={width - lpad * 3}  y={i * height + (i+1) * tpad} fontSize={12} ></Text>
                {/* <Rect name="bomb"  fill={"white"} width={width/10} height={height/10} x={width - (width/10) - lpad}  y={bottom + tpad/2} ></Rect> */}
                {(playerBox.hasBomb && playerBox.bomb) && (<Image image={playerBox.bomb} ref={(n) => {if (n) elements.bombImage = n}} name="bomb" x={width - (width/10) - lpad/2}  y={bottom - tpad/2} height={25} width={25}/>)} 
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={i * height + (i+1) * tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${playerBox.money}`} fill={"white"} x={lpad}  y={mid} fontSize={12} ></Text>
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}} src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
            </Group>
    }
    
}

function DemoPlayback({file, map}:{file:String, map:String}){
        const bomb_source = '/equipment/c4.svg'
        const weaponImageCacheRef = useRef<Map<string, HTMLImageElement>>(new Map());
        const WEAPON_IDS = [
            1, 2, 3, 4, 5, 6, 7, 8, 9, 10,       // Pistols
            101, 102, 103, 104, 105, 106, 107,   // SMGs
            201, 202, 203, 204, 205, 206,         // Heavy
            301, 302, 303, 304, 305, 306, 307, 308, 309, 310, 311, // Rifles
            401, 404, 405,                       // Zeus, Bomb, Knife
            501, 502, 503, 504, 505, 506         // Grenades
        ];
        const [bombSvg] = useImage(bomb_source);
        const [stats, setStats] = useState<MatchEvents>()
        const [size, setSize] = useState({ width: 0, height: 0 });
        const [stageDim, setStageDim] = useState({ width: 0, height: 0 });
        const [round, setRound] =useState(1);
        const [isPlaying, setPlaying] = useState<PlaybackState>({playing: false, round_no:1, tick_no: 0});
        const playbackContainer = useRef<HTMLDivElement>(null);
        const round_begin_ticks = useRef<number[]>([]);
        const grenadeCache = useRef<Map<string, EntityInfo>>(new Map());
        const grenadeLife = useRef<Map<string, EntityLife>>(new Map())
        const tickRef = useRef(isPlaying.tick_no);
        const playerRef = useRef<Map<string, Konva.Group>>(null);
        const hudRef = useRef<Map<string, Konva.Group>>(new Map());
        const hudCacheRef = useRef<Map<string, PlayerHudCache>>(new Map());
        const progressRef = useRef<HTMLInputElement>(null);
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
            
            WEAPON_IDS.forEach((v) => {
                const src = type_to_svg(v)
                const img = new window.Image();
                img.src = src
                if (img != null) weaponImageCacheRef.current.set(src, img!)
            })
            return () => { ignore = true}
        }, [file, round])
        // Handle Responsive Resizing
        useEffect(() => {

             const updateSize = () => {
                    if (playbackContainer.current) {
                        const parentElement = playbackContainer.current.parentElement;
                    if (!parentElement) return;

                    const availableHeight = parentElement.offsetHeight;
                    const availableWidth = parentElement.offsetWidth;

                    // Use the smaller dimension to keep it a square
                    const side = Math.min(availableHeight, availableWidth);                    
                    setSize({ width: side, height: side });
                    setStageDim({width:playbackContainer.current?.getBoundingClientRect().width!, height:playbackContainer.current?.getBoundingClientRect().height!})
                }
            };

            const observer = new ResizeObserver(updateSize);
            if (playbackContainer.current) {
                observer.observe(playbackContainer.current);
            }

            updateSize(); 


            return () => observer.disconnect();
        }, []);
        useEffect(() => {
        const TARGET_TICK_RATE = 64; 
        const MS_PER_TICK = 1000 / TARGET_TICK_RATE;
        let accumulatedTime = 0;
        const anim = new Konva.Animation((frame) => {
            if (!frame) return
            if (!isPlaying.playing) {
                tickRef.current = isPlaying.tick_no
                return
            }
            accumulatedTime += frame.timeDiff;
            while (accumulatedTime >= MS_PER_TICK) {
            if (tickRef.current === 0) {
                tickRef.current = isPlaying.tick_no;
            }
            tickRef.current += 1;
            accumulatedTime -= MS_PER_TICK;
            }
            if (progressRef.current) {
                progressRef.current.value = tickRef.current.toString();
                
            }
            if (playerRef.current != null){
                if (tickRef.current === 0) {
                    tickRef.current = isPlaying.tick_no;
                }
                playerRef.current?.forEach((group_object, itemid) => {
                    const x = group_object!.getAttr("name")
                        if (x == "player"){    
                            if (playbackRef.current!.player_pos.has(tickRef.current)){
                                const positions = playbackRef.current!.player_pos.get(tickRef.current)!.get(itemid)
                                group_object.getChildren().forEach((g) => {
                                // Positions become null means that the player died in the server because we stopped recording them
                                // TODO Maybe change it to an X if I can?
                                if(positions == null){
                                        g.x(-1000);
                                        g.y(-1000)
                                } else {
                                    if (g.className == "Circle") {
                                        g.x(positions!.vector.X);
                                        g.y(positions!.vector.Y)
                                    } else {
                                        g.x(positions!.vector.X+5);
                                        g.y(positions!.vector.Y-3)
                                    }
                                }})
                            }
                        } else {
                            let lastInfo = grenadeCache.current.get(itemid)
                            if (lastInfo != null){
                                if (lastInfo.kind == "GRENADE"){
                                    if (!playbackRef.current!.grenade_pos.has(tickRef.current)) return
                                    const gren_pos = playbackRef.current!.grenade_pos.get(tickRef.current)!.get(itemid)
                                    if (gren_pos != null){
                                        if (lastInfo.status == gren_pos.status){
                                            switch (gren_pos.status){
                                            case "FLYING":
                                                group_object.getChildren().forEach((g) => {
                                                if (g.className == "Circle") {
                                                    g.x(gren_pos!.vector.X);
                                                    g.y(gren_pos!.vector.Y)  
                                                } else {
                                                    g.x(gren_pos!.vector.X+5);
                                                    g.y(gren_pos!.vector.Y-3)
                                                }

                                            })
                                            break;
                                            case "BLOOMED":
                                                break;
                                            }
                                        } else {
                                            switch(gren_pos.status){
                                               // FLYING -> BLOOMED
                                                case "BLOOMED":
                                                    group_object.getChildren().forEach((g) => {
                                                        if (g.className == "Circle") {
                                                            const circle = g as Konva.Circle
                                                            circle.radius(size.height * .035);
                                                        }
                                                    })
                                                    lastInfo.status = "BLOOMED"
                                                    lastInfo.lastTickUpdate = tickRef.current
                                                    grenadeCache.current.set(itemid, lastInfo)
                                                break;
                                                // BLOOMED -> EXPIRED || FLYING -> LANDED || EXPIRED
                                                case "EXPIRED":
                                                case "LANDED":
                                                    group_object.destroyChildren()     
                                                    grenadeCache.current.delete(itemid)         
                                                    playerRef.current?.delete(itemid)
                                                    getPlayerRef().delete(itemid)
                                                break;
                                            }
                                        }
                                    }
                                } else if (lastInfo.kind == "BOMB"){
                                    if (!playbackRef.current!.grenade_pos.has(tickRef.current)) return
                                    const gren_pos = playbackRef.current!.grenade_pos.get(tickRef.current)!.get(itemid)
                                    if (gren_pos != null){
                                        if (lastInfo.status != gren_pos.status){
                                            
                                            switch (gren_pos.status){
                                                // DROPPED -> GRABBED
                                                case "GRABBED":
                                                    group_object.destroyChildren()
                                                    playerRef.current!.delete(itemid)
                                                    grenadeCache.current.delete(itemid)
                                                    getPlayerRef().delete(itemid)
                                                break;
                                                // PLANTED -> DEFUSED
                                                case "DEFUSED":
                                                    const image: Konva.Image | undefined= group_object.findOne((n:Konva.Node) => {
                                                        return n.className == 'Image'
                                                    })
                                                    
                                                    if (image) {
                                                        image.hue(240)
                                                        image.saturation(100)
                                                        image.value(255)
                                                        image.clearCache()
                                                        image.cache()
                                                    }
                                                    lastInfo.lastTickUpdate = tickRef.current
                                                    lastInfo.status = "DEFUSED"
                                                    grenadeCache.current.set(itemid, lastInfo)
                                                break;
                                                default:
                                                return
                                            }
                                        }
                                    }
                                }
                            }
                            if (group_object.hasName("FIRE")){
                                if (!playbackRef.current!.fire_vertices.has(tickRef.current)) return
                                const fire_info = playbackRef.current!.fire_vertices.get(tickRef.current)!.get(itemid)
                                if (fire_info != null){
                                    if (group_object.hasName(fire_info.status)){
                                        switch (fire_info.status){
                                            case "STARTING":
                                                // BEGINS AS CIRCLE. 
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
                                            break;
                                            case "SPREADING":
                                                const curr_vertices = parseInt(group_object.name()[group_object.name().length-1])
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
                                            break;
                                        }
                                    } else {
                                       switch(fire_info.status){
                                        // SPREADING -> ENDING
                                            case "ENDING":
                                                group_object.destroyChildren()
                                                playerRef.current?.delete(itemid)
                                                getPlayerRef().delete(itemid)
                                            break;
                                            // STARTING -> SPREADING
                                            case "SPREADING":
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
                                            break;
                                       }
                                    }
                                }
                            }  
                        }
                })
                // Check if a grenade needs to be created   
                if (playbackRef.current!.grenade_pos.has(tickRef.current)) {
                    const map = getPlayerRef()
                    const tickMap = playbackRef.current!.grenade_pos.get(tickRef.current)
                    tickMap?.forEach((state, id) => {
                        if (map.has(id)){
                            return;
                        } else {
                            if (state.status == "EXPIRED" || state.status == "LANDED" || state.status == "ENDING" || state.status == "GRABBED") { return } 
                            const mainGroup = layerRef.current!.findOne("#mainPlayer") as Konva.Group
                            if (grenadeCache.current == null){
                                grenadeCache.current = new Map();
                            }
                            if (state.grenade == "BOMB") {
                                // console.log("BOMB NEEDS TO BE ADDED TO MAP")
                                let bomb = new Konva.Group({name:`BOMB ${state.status}`, id:id})
                                let rect = new Konva.Image({
                                      filters:[Konva.Filters.HSV],  image:bombSvg, x: state.vector.X-25/2, y: state.vector.Y-25/2, width:25, height:25,
                                })
                                rect.cache()
                                let cache:EntityInfo = {
                                    kind: "BOMB", status: state.status, id: id, lastTickUpdate:tickRef.current
                                }
                                grenadeCache.current.set(id, cache)
                                switch (state.status) {
                                    case "DROPPED":
                                        bomb.add(rect)
                                        rect.stroke("WHITE")
                                        map.set(id, bomb)
                                        mainGroup.add(bomb)
                                    break;
                                    case "PLANTED":
                                        // rect.stroke("red")
                                        // rect.fill("red")
                                        // 70 is Red??
                                        rect.hue(70)
                                        rect.saturation(100)
                                        rect.value(255)
                                        rect.clearCache()
                                        rect.cache()
                                        bomb.add(rect)
                                        map.set(id, bomb)
                                        mainGroup.add(bomb)
                                    break;
                                    default:
                                        console.log(`${state.grenade} ${state.status} has not been handled yet.`)
                                    return;
                                }
                                
                            } else {
                                let gren = new Konva.Group({name:`GRENADE ${state.grenade} ${state.status}`, id:id})
                                let circl = new Konva.Circle({
                                        x: state.vector.X, y: state.vector.Y, radius: size.width * .01 , fill:"white"
                                    })
                                let label = new Konva.Text({
                                        x: state.vector.X + 5, y: state.vector.Y - 3, text: state.grenade,
                                        fill:"white" , fontSize:10 
                                })
                                let cache:EntityInfo = {
                                    kind: "GRENADE", status: state.status, id: id, lastTickUpdate:tickRef.current
                                }
                                grenadeCache.current.set(id, cache)
                                gren.add(circl, label)
                                map.set(id, gren)
                                mainGroup.add(gren)
                            }
                            
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
                            let fire = new Konva.Group({name:`FIRE ${state.status}`})
                            let circl = new Konva.Circle({
                                x : state.vertices[0].X , y: state.vertices[0].Y, radius: size.height * .02, fill:"orange"
                            })
                            const mainGroup = layerRef.current!.findOne("#mainPlayer") as Konva.Group
                            fire.add(circl)
                            map.set(id, fire)
                            mainGroup.add(fire)
                        }
                    })
                }
                if (hudRef.current != null) {
                    // console.log("IN HUD")
                    // console.log(hudRef.current)
                    hudRef.current!.forEach((g, id) => {
                        if (playbackRef.current!.player_pos.has(tickRef.current)){
                            const p = playbackRef.current!.player_pos.get(tickRef.current)?.get(id)
                            if (p != null){
                                // console.log("CHANGING")
                                let t = hudCacheRef.current.get(id)
                                let y = t!.activeWep.y()
                                t!.statsText.text(`Kills: ${p.kills}, Assists: ${p.assists}, Deaths: ${p.deaths} $${p.dinero}`)
                                // text.text(`${p.active_weapon} GS: HE,F,F,M 30/4`)
                                t!.hpBar.width(p.hp/100 * (stageDim.width-size.width)/2)
                                t!.hpText.text(`${p.hp}`)
                                const src = type_to_svg(p.active_weapon)
                                const cacheImg = weaponImageCacheRef.current.get(src)
                                if (cacheImg && t!.activeWep.image() !== cacheImg){
                                    t!.activeWep.image(cacheImg)
                                    if (src == 'world.svg'){
                                        console.log(p.active_weapon)
                                    }
                                }
                                if (p.hasBomb && t!.bombImage == null){
                                    const w = (stageDim.width-size.width)/2
                                    t!.bombImage = new Konva.Image({image:bombSvg, height:25, width:25, y:y-5, x:(w-(w/10)-5)})
                                    g.add(t!.bombImage)
                                } else {
                                    if (t!.bombImage && !p.hasBomb){
                                        t!.bombImage.hide()
                                    }
                                }

                            } else {
                                let t = hudCacheRef.current.get(id)
                                 t!.hpText.text(`0`)
                                 t!.hpBar.width(0)
                                if (t?.bombImage != null){
                                    t!.bombImage.destroy()
                                }
                                
                            }
                        } 
                    })
                }
            }

            

        }, layerRef.current);
        anim.start()       
        return () => {
            anim.stop()
        };
        }, [isPlaying.playing, round]);
        // ID, NAME, X, Y, SIDE, WEAPON
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
                return (originY-y)/mapScale * stageDim.height/1024
            }
            const pos = Array.from(Object.entries(stats.round_events.player_positions))
            const grenades = Array.from(Object.entries(stats.round_events.grenade_events))
            const fire_vertices = Array.from(Object.entries(stats.round_events.fire_events))
            let tick_map:PlayBackRef = {
                player_pos: new Map<number, Map<string, PlayerState>>(),
                grenade_pos: new Map<number, Map<string, GrenadeState>>(),
                fire_vertices: new Map<number, Map<string, FireState>>(),
                player_info: new Map<string, PlayerInformation>()
            };
            if (grenadeLife.current == null){
                grenadeLife.current = new Map();
            }
            // Assume that each tick in PlayerState also exists in Grenade
            pos.forEach(([tick, playervec],i) => {
                const info = Array.from(Object.entries(playervec))
                const player_pos = new Map<string, PlayerState>()
                info.forEach(([playerid, state]) =>{
                    const place:MapCoordinate = {
                        X:newX(state.vector.X), Y:newY(state.vector.Y)
                    }
                    const player_state: PlayerState = {
                        vector: place, active_weapon: state.active_weapon, hp:state.hp, kills: state.kills, assists:state.assists, deaths:state.deaths, armor:state.armor, dinero:state.dinero,
                        action: state.action, hasBomb: state.hasBomb
                    }
                    player_pos.set(playerid, player_state)
                })
                
                tick_map.player_pos.set(Number(tick), player_pos)
                if (i ==0) {
                    tick_map.player_pos.set(0, player_pos)
                    round_begin_ticks.current[0] = (Number(tick))
                    tickRef.current = Number(tick)
                    
                }
                if (i == (pos.length-1)){
                    round_begin_ticks.current[1] = (Number(tick))
                }
            });
            grenades.forEach(([tick, grenadeEvent]) => {
                const grenade_info = Array.from(Object.entries(grenadeEvent))
                const grenade_pos = new Map<string, GrenadeState>();
                grenade_info.forEach(([grenid, grenstate]) => {
                    const place:MapCoordinate = {
                        X:newX(grenstate.vector.X), Y:newY(grenstate.vector.Y)
                    }
                    const grenade_state: GrenadeState = {
                        vector: place, thrownBy: grenstate.thrownBy, thrownById: grenstate.thrownById, grenade:grenstate.grenade, status:grenstate.status
                    }
                    const state = grenadeLife.current.get(grenid)
                    if (state == null){
                       const ent:EntityLife = {
                            grenadeString: grenstate.grenade, lastSavedState: grenstate.status, flyingStart: Number(tick), flyingExpire: -1
                       } 
                       grenadeLife.current.set(grenid, ent)
                    } else {
                        if (state.lastSavedState != grenstate.status){
                            switch(grenstate.status){
                                // FLYING -> BLOOMED
                                case "BLOOMED":
                                    state.lastSavedState = "BLOOMED"
                                    state.flyingExpire = Number(tick)
                                    state.bloomedStart = Number(tick)
                                break;
                                case "EXPIRED":
                                    // BLOOMED -> EXPIRED
                                    state.lastSavedState = "EXPIRED"
                                    if (grenstate.grenade == "Smoke Grenade"){
                                        state.bloomExpiry = Number(tick)
                                    } 
                                    // FLYING -> EXPIRED
                                    else {
                                        state.flyingExpire = Number(tick)
                                    }
                                    grenadeLife.current.set(grenid, state)
                                break;
                                // FLYING -> LANDED 
                                // THIS IS MOLLIES
                                case "LANDED":
                                    state.lastSavedState = "LANDED"
                                    state.flyingExpire = Number(tick)
                                    grenadeLife.current.set(grenid, state)
                                break;

                            }
                        }
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
                playbackRef.current!.player_info.set(playerid, playername)
            })
            
        }
        const freeSpace:number = (stageDim.width-size.width)/2
    return <>
        <div id="playbackGrid" >
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
            <div id="playbackMap" ref={playbackContainer}>
                <Stage  width={stageDim.width} height={stageDim.height}>
                    <Layer ref={layerRef}   >
                        <Group x={freeSpace} id={"mainPlayer"}>
                            <URLImage src={`/overviews/${map}.jpg`}  width={size.width} height={stageDim.height}></URLImage>     
                       
                                { stats && 
                                    Array.from(playbackRef.current!.player_pos.get(0)!.entries()).map(([playerid, ps], i) => {
                                      const info = playbackRef.current!.player_info.get(playerid)!
                                      const side = info.side
                                      const color = side == 2 ? "orange" : "blue"
                                      if (side != null){
                                        return (
                                            <Group  key={i} name={"player"} ref={(node) =>{
                                                    const map = getPlayerRef();
                                                    if (node != null) {
                                                        map.set(playerid, node)
                                                    }
                                                    return () => {map.delete(playerid)}
                                                }}>
                                            <Circle
                                                x={ps!.vector.X}
                                                y={ps!.vector.Y}
                                                fill={color}
                                                radius={5}
                                            />
                                            <Text 
                                                text={info.name} 
                                                x={ps!.vector.X + 5} 
                                                y={ps!.vector.Y - 3} 
                                                fill="white" 
                                                fontSize={10} 
                                            />
                                        </Group>
                                        )
                                      }
                                    })
                                } 
                        </Group>              
                           {/* These are the player info boxes  */}
                       <Group x={0} >
                            {   stats &&
                                Array.from(playbackRef.current!.player_pos.get(0)!.entries()).filter(([playerid, s]) => {
                                    const info = playbackRef.current!.player_info.get(playerid)!
                                    const side = info.side
                                    return side == 2
                                }).map(([playerid, ps], i) => {
                                    const name = playbackRef.current!.player_info.get(playerid)!.name
                                    const info:PlayerBox = {
                                        width:freeSpace, height:size.height/10, tpad:10, lpad:10, i:i, name:name, playerid:playerid,weapon:ps.active_weapon, hp:ps.hp, hasBomb:ps.hasBomb, bomb:bombSvg, money: ps.dinero
                                    }
                                    return playerBoxInfo({playerBox:info, hud:hudRef.current, cache:hudCacheRef.current})
                                })
                            }
                        </Group>
                        <Group x={stageDim.width-freeSpace} >
                            {   stats &&
                                Array.from(playbackRef.current!.player_pos.get(0)!.entries()).filter(([playerid, s]) => {
                                    const info = playbackRef.current!.player_info.get(playerid)!
                                    const side = info.side
                                    return side != 2
                                }).map(([playerid, ps], i) => {
                                    const name = playbackRef.current!.player_info.get(playerid)!.name
                                    const info:PlayerBox = {
                                        width:freeSpace, height:size.height/10, tpad:10, lpad:10, i:i, name:name, playerid:playerid,weapon:ps.active_weapon, hp:ps.hp, hasBomb:ps.hasBomb, bomb:bombSvg, money: ps.dinero
                                    }
                                    return playerBoxInfo({playerBox:info, hud:hudRef.current, cache:hudCacheRef.current})
                                })
                            }
                        </Group>
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
                        console.log(`BEFORE ANY ACTION TICK:${tickRef.current}`)
                        // setPlaying({...isPlaying, tick_no:tickRef.current, playing: !isPlaying.playing}) 
                        // Going from Playing to Pause
                        if (isPlaying.playing == true){
                            console.log(`PAUSE TICK:${tickRef.current}`)
                            setPlaying({...isPlaying, tick_no:tickRef.current, playing: false})
                        } else {
                            tickRef.current = isPlaying.tick_no
                            console.log(`UNPAUSE TICK:${tickRef.current}`)
                            setPlaying({...isPlaying, tick_no:tickRef.current, playing: true})
                            tickRef.current = isPlaying.tick_no
                        }
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
                {stats && 
                    <input id="playback-slider" onChange={(e) => {
                        const target = parseInt(e.target.value, 10)
                        tickRef.current = target;
                        // setPlaying((prev) => ({ ...prev, tick_no: target }));
                    }} ref={progressRef} type="range" style={{width: "100%"}}  min={tickRef.current}  value={tickRef.current} max={round_begin_ticks.current[1]}/>
                }
                
            </div>
            <div className="rounds">
                <ul style={{justifyContent:"center", marginTop:"10px"}}>
                {stats && 
                        Array.from({length:stats.rounds}, (_, i) => i+1).map((n, j) => {
                            return <li key={j} onClick={() => {
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
                                hudRef.current!.forEach((g) => { 
                                    let node = g.findOne((n:Konva.Node) => {
                                        return n.getAttr("name") == "hpbar"
                                    })
                                    const hp = node as Konva.Rect
                                    hp.width((stageDim.width-size.width)/2)
                                    node = g.findOne((n:Konva.Node) => {
                                        return n.getAttr("name") == "hp"
                                    })
                                    let text = node as Konva.Text
                                    text.text(`${100}`)
                                    node = g.findOne((n:Konva.Node) => {
                                        return n.getAttr("name") == "bomb"
                                    })
                                    if (node != null){
                                        node.destroy()
                                    }

                                });
                                setPlaying({tick_no:round_begin_ticks.current[1], playing:false, round_no: n})
                                setRound(n);
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