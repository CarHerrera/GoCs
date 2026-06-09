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
    blind_dur: number
    primary:number
    secondary:number
    slot1: number
    slot2: number
    slot3: number
    slot4: number
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
    grenade?: Konva.Circle 
    text?: Konva.Text 
    bomb?: Konva.Image 
}
interface EntityLife{
    grenadeString: string
    lastSavedState: string
    flyingStart: number
    bloomedStart?: number
    bloomExpiry?: number
    flyingExpire: number
}
interface FireLife {
    circle?: Konva.Circle
    state: string
    spread?: Konva.Shape
    vertices: number
}
interface PlayerBox {
    width: number
    height: number
    tpad: number
    lpad: number
    i: number
    name: string
    playerid: string
    ps: PlayerState
}
interface PlayerHudCache {
    hpBar: Konva.Rect;
    hpText: Konva.Text;
    statsText: Konva.Text;
    activeWep: Konva.Image;
    bombImage: Konva.Image | null;
    priamry: Konva.Image | null;
    secondary: Konva.Image | null;
    slot1: Konva.Image | null;
    slot2: Konva.Image | null;
    slot3: Konva.Image | null;
    slot4: Konva.Image | null;
}
interface PlayerPos {
    circle: Konva.Circle
    Name: Konva.Text
}
interface PlayBackCache{
    PlayerCache: Map<String, PlayerPos>
    GrenadeCache: Map<String,EntityInfo>
    FireCache: Map<String, FireLife>
    GrenadeLife: Map<String, EntityLife>
    playback: PlayBackRef | null
    Tick: number
}
type WeaponIconProps = Omit<Konva.ImageConfig, 'image'> & { src: string }
function WeaponIcon({src, ...rest}: WeaponIconProps){
    const [img] = useImage(src, 'anonymous')
    return img ? <Image {...rest} image={img}/> : null
}
function playerBoxInfo({playerBox, hud, cache}:{playerBox:PlayerBox, hud: Map<string, Konva.Group>, cache:Map<string, PlayerHudCache>}){
    let color ="white"
    const {width, height, tpad, lpad, i, name, playerid, ps} = playerBox
    
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
    const active_weapon = type_to_svg(ps.active_weapon)
    // const primary = type_to_svg(ps.primary)
    const secondary = type_to_svg(ps.secondary)
    // const slot1 = type_to_svg(ps.slot1)
    // const slot2 = type_to_svg(ps.slot2)
    // const slot3 = type_to_svg(ps.slot3)
    // const slot4 = type_to_svg(ps.slot4)
    const wep_width = 60
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
                {(ps.hasBomb) && 
                (<WeaponIcon name="bomb" ref={(n: Konva.Image) => {if (n) elements.bombImage = n}} src={type_to_svg(404)} x={width - (width/10) - lpad/2}  y={bottom -tpad/2} height={25} width={25}></WeaponIcon>)
                } 
                {/* (<Image image={playerBox.bomb} ref={(n) => {if (n) elements.bombImage = n}} name="bomb"  x={width - (width/10) - lpad/2}  y={bottom -tpad/2   } height={25} width={25} />) */}
                <Text name="hp" ref={(n) => {if (n) elements.hpText = n}} text={`${ps.hp}`} fill={"white"} x={width - lpad * 3}  y={tpad} fontSize={12} ></Text>
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${ps.dinero}`} fill={"white"} x={lpad}  y={bottom/2 + tpad} fontSize={12} ></Text>
                {/* <Text width={width} name="inv" text={`${weapon} HE,F,F,M 30/4`} fill={"white"} x={lpad}  y={bottom} fontSize={12} ></Text> */}
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}}  src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
                {ps.secondary != ps.active_weapon && 
                    <WeaponIcon src={secondary} x={70} y={bottom -tpad}></WeaponIcon>
                }
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
                <Text name="hp" ref={(n) => {if (n) elements.hpText = n}}  text={`${ps.hp}`} fill={"white"} x={width - lpad * 3}  y={i * height + (i+1) * tpad} fontSize={12} ></Text>
                {(ps.hasBomb) && 
                (<WeaponIcon name="bomb" ref={(n: Konva.Image) => {if (n) elements.bombImage = n}} src={type_to_svg(404)} x={width - (width/10) - lpad/2}  y={bottom -tpad/2} height={25} width={25}></WeaponIcon>)} 
                                {/* (<Image image={ps.bomb} ref={(n) => {if (n) elements.bombImage = n}} name="bomb" x={width - (width/10) - lpad/2}  y={bottom - tpad/2} height={25} width={25}/>) */}
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={i * height + (i+1) * tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${ps.dinero}`} fill={"white"} x={lpad}  y={mid} fontSize={12} ></Text>
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}} src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
                {ps.secondary != ps.active_weapon && 
                    <WeaponIcon ref={(n:Konva.Image) => {if (n) elements.secondary = n}} src={secondary} x={wep_width + 10} y={bottom -tpad}></WeaponIcon>
                }
            </Group>
    }
    
}

function RedrawAtTicK(cache:PlayBackCache){
    const {PlayerCache, GrenadeCache, FireCache, GrenadeLife, playback, Tick} = cache

    if (playback != null){
        console.log(playback.player_pos.has(Tick))
        console.log(playback.grenade_pos.has(Tick))
        console.log(playback.fire_vertices.has(Tick))
    }
}
function DemoPlayback({file, map}:{file:String, map:String}){
        const bomb_source = '/equipment/c4.svg'
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
        const weaponImageCacheRef = useRef<Map<string, HTMLImageElement>>(new Map());
        const playerPosCacheRef = useRef<Map<string, PlayerPos>>(new Map())
        const grenadeCache = useRef<Map<string, EntityInfo>>(new Map());
        const grenadeLife = useRef<Map<string, EntityLife>>(new Map())
        const fireVertices = useRef<Map<string, FireLife>>(new Map());
        const tickRef = useRef(isPlaying.tick_no);
        const playingRef = useRef(false)
        const hudRef = useRef<Map<string, Konva.Group>>(new Map());
        const hudCacheRef = useRef<Map<string, PlayerHudCache>>(new Map());
        const progressRef = useRef<HTMLInputElement>(null);
        // ROUND NO -> TICK NO -> PLAYBACK REF
        const playbackRef = useRef<PlayBackRef>(null);
        const layerRef = useRef<Konva.Layer>(null)
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

        useEffect(() => {
            if (progressRef.current) {
                progressRef.current.value = tickRef.current.toString();
            }
        }, [stats]);

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
                return
            }
            accumulatedTime += frame.timeDiff;
            while (accumulatedTime >= MS_PER_TICK) {
            // if (tickRef.current === 0) {
            //     tickRef.current = isPlaying.tick_no;
            // }
            tickRef.current += 1;
            accumulatedTime -= MS_PER_TICK;
            }
            if (progressRef.current) {
                progressRef.current.value = tickRef.current.toString();
            }
            if (playbackRef.current != null){
                if (playbackRef.current.player_pos.has(tickRef.current)){
                    playerPosCacheRef.current.forEach((cache, playerid) => {
                        const ps = playbackRef.current!.player_pos.get(tickRef.current)!.get(playerid)
                        if (ps != null){
                            cache.circle.x(ps!.vector.X)
                            cache.circle.y(ps!.vector.Y)
                            cache.Name.x(ps!.vector.X + 5)
                            cache.Name.y(ps!.vector.Y - 3)
                        } else {
                            cache.circle.x(-1000)
                            cache.circle.y(-1000)
                            cache.Name.x(-1000)
                            cache.Name.y(-1000)
                        }
                    })
                }
                if (playbackRef.current!.grenade_pos.has(tickRef.current)) {
                    const tickMap = playbackRef.current!.grenade_pos.get(tickRef.current)
                    tickMap?.forEach((state, id) => {
                        if (grenadeCache.current.has(id)){
                            const gren = grenadeCache.current.get(id)
                            if (gren!.kind == "GRENADE"){
                                if (gren!.status == state.status){
                                    switch (state.status){
                                        case "FLYING":
                                            gren!.grenade!.x(state.vector.X) 
                                            gren!.grenade!.y(state.vector.Y) 
                                            gren!.text!.x(state.vector.X + 5)
                                            gren!.text!.y(state.vector.Y - 3)
                                        break;
                                        case "BLOOMED":
                                            break;
                                    }
                                } else {
                                     switch(state.status){
                                               // FLYING -> BLOOMED
                                        case "BLOOMED":
                                            gren!.grenade!.radius(size.height * .035)
                                            gren!.text!.hide()
                                            gren!.status = "BLOOMED"
                                            gren!.lastTickUpdate = tickRef.current
                                            grenadeCache.current.set(id, gren!)
                                        break;
                                        // BLOOMED -> EXPIRED || FLYING -> LANDED || EXPIRED
                                        case "EXPIRED":
                                        case "LANDED":
                                            // Maybe don't delete?
                                            gren!.grenade!.hide()
                                            gren!.text!.hide()
                                        break;
                                    }
                                }
                            } else if (gren!.status == "BOMB"){
                                if (gren!.status != state.status){
                                    switch (state.status){
                                        case "GRABBED":
                                            gren!.bomb!.hide()
                                            gren!.bomb!.clearCache()
                                            gren!.bomb!.cache()
                                        break;
                                        case "DEFUSED":
                                            gren!.bomb!.hue(240)
                                            gren!.bomb!.saturation(100)
                                            gren!.bomb!.value(255)
                                            gren!.bomb!.clearCache()
                                            gren!.bomb!.cache()
                                            gren!.lastTickUpdate = tickRef.current
                                            gren!.status = "DEFUSED"
                                            grenadeCache.current.set(id, gren!)
                                        break;
                                        default:
                                        return
                                    }
                                }
                            }
                        } else {
                            if (state.status == "EXPIRED" || state.status == "LANDED" || state.status == "ENDING" || state.status == "GRABBED") { return } 
                            const mainGroup = layerRef.current!.findOne("#mainPlayer") as Konva.Group

                            if (state.grenade == "BOMB") {
                                // console.log("BOMB NEEDS TO BE ADDED TO MAP")
                                let bomb = new Konva.Group({name:`BOMB ${state.status}`, id:id})
                                let rect = new Konva.Image({
                                      filters:[Konva.Filters.HSV],  image:bombSvg, x: state.vector.X-25/2, y: state.vector.Y-25/2, width:25, height:25,
                                })
                                rect.cache()
                                let cache:EntityInfo = {
                                    kind: "BOMB", status: state.status, id: id, lastTickUpdate:tickRef.current, bomb:rect
                                }
                                grenadeCache.current.set(id, cache)
                                switch (state.status) {
                                    case "DROPPED":
                                        bomb.add(rect)
                                        rect.stroke("WHITE")
                                        mainGroup.add(bomb)
                                    break;
                                    case "PLANTED":
                                        rect.hue(70)
                                        rect.saturation(100)
                                        rect.value(255)
                                        rect.clearCache()
                                        rect.cache()
                                        bomb.add(rect)
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
                                    kind: "GRENADE", status: state.status, id: id, lastTickUpdate:tickRef.current, grenade: circl, text: label
                                }
                                grenadeCache.current.set(id, cache)
                                gren.add(circl, label)
                                
                                mainGroup.add(gren)
                            }
                            
                        }
                    })
                }

                if(playbackRef.current!.fire_vertices.has(tickRef.current)){
                    const mainGroup = layerRef.current!.findOne("#mainPlayer") as Konva.Group
                    const tickMap = playbackRef.current!.fire_vertices.get(tickRef.current)
                    tickMap?.forEach((state, id) => {
                        if (fireVertices.current.has(id)){  
                            const fireInfo = fireVertices.current.get(id)!
                            if (fireInfo.state == state.status){
                                switch (state.status){
                                    case "SPREADING":
                                        if (fireInfo.vertices != state.vertices.length){
                                            fireInfo.spread!.setAttr("customVertices", state.vertices)
                                            fireInfo.vertices = state.vertices.length
                                            fireVertices.current!.set(id, fireInfo)
                                        }
                                        if (fireInfo.circle != null && fireInfo.circle!.isVisible()){
                                                fireInfo.circle!.hide()
                                        }
                                    break;
                                }
                            } else {
                                if (!fireInfo.spread && state.vertices.length > 1){
                                    const spread:Konva.Shape = new Konva.Shape({
                                        fill:"orange",
                                        customVertices: state.vertices,
                                        sceneFunc(con, shape) {
                                            con.beginPath()
                                            const vertes = shape.getAttr('customVertices')
                                            vertes.forEach((vertex:MapCoordinate, i:number) => {
                                                if (i == 0){
                                                    con.moveTo(vertex.X, vertex.Y)
                                                    
                                                } else {
                                                    con.lineTo(vertex.X, vertex.Y)
                                                }
                                            })
                                            con.closePath()
                                            con.fillStrokeShape(shape)
                                        },
                                    })
                                    mainGroup.add(spread);
                                    fireInfo.vertices = state.vertices.length
                                    fireInfo.state = state.status
                                    fireInfo.spread = spread
                                }
                                switch(state.status){
                                    case "ENDING":
                                        fireInfo.spread!.hide()
                                        fireInfo.state = "ENDING"
                                    break;
                                    case "SPREADING":
                                        if (state.vertices.length >1 && fireInfo.vertices != state.vertices.length){
                                            fireInfo.spread!.setAttr("customVertices", state.vertices)
                                            fireInfo.vertices = state.vertices.length
                                        }
                                        break;
                                }
                                if (fireInfo.circle != null && fireInfo.circle!.isVisible()){
                                    fireInfo.circle!.hide()
                                }
                                fireVertices.current.set(id, fireInfo)
                            }
                        } else {
                            if (state.status == "ENDING") {return}
                            let fire = new Konva.Group({name:`FIRE ${state.status}`})
                            let circl = new Konva.Circle({
                                x : state.vertices[0].X , y: state.vertices[0].Y, radius: size.height * .02, fill:"orange"
                            })
                            const mainGroup = layerRef.current!.findOne("#mainPlayer") as Konva.Group
                            fire.add(circl)
                            const fireEnt: FireLife = {
                                circle: circl, state: "STARTING", vertices: 1
                            }
                            fireVertices.current.set(id, fireEnt)
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
                                let t = hudCacheRef.current.get(id)
                                let y = t!.activeWep.y()
                                t!.statsText.text(`Kills: ${p.kills}, Assists: ${p.assists}, Deaths: ${p.deaths} $${p.dinero}`)
                                t!.hpBar.width(p.hp/100 * (stageDim.width-size.width)/2)
                                t!.hpText.text(`${p.hp}`)
                                const grenades = [p.slot1, p.slot2, p.slot3, p.slot4]
                                const src = type_to_svg(p.active_weapon)
                                const cacheImg = weaponImageCacheRef.current.get(src)
                                let width = t!.activeWep.width()
                                if (cacheImg && t!.activeWep.image() !== cacheImg){
                                    t!.activeWep.image(cacheImg)
                                    width = cacheImg.width
                                }
                                let nextXOffset = t!.activeWep.x() + width + 10;
                                const sec = weaponImageCacheRef.current.get(type_to_svg(p.secondary))
                                if (t!.secondary == null ){
                                    if (p.secondary !== p.active_weapon && t!.secondary !== 0) {
                                        t!.secondary = new Konva.Image({image:sec, x:(nextXOffset), y:y })
                                        g.add(t!.secondary)
                                        nextXOffset += t!.secondary.width()
                                    } 
                                } else {
                                    if (p.secondary == p.active_weapon || p.secondary == 0) {
                                        t!.secondary.hide()
                                        // width -= t!.secondary.width()
                                    } else {
                                        t!.secondary.image(sec);
                                        t!.secondary.x(nextXOffset);
                                        if (!t!.secondary.isVisible()){
                                            t!.secondary.show()
                                        }
                                        nextXOffset += (sec?.width || t!.secondary.width()) + 10;
                                    }
                                }
                                
                                grenades.forEach((id, i) => {
                                    const slotKey = `slot${i + 1}` as keyof typeof t;
                                    if (id !== 0) {
                                    const grenImg = weaponImageCacheRef.current.get(type_to_svg(id));
                                    let slotNode = t![slotKey] as Konva.Image | undefined;

                                    if (!slotNode) {
                                        // First time seeing this item slot: instantiate it
                                        const newGrenNode = new Konva.Image({ image: grenImg, x: nextXOffset, y: y });
                                        (t as any)[slotKey] = newGrenNode;
                                        g.add(newGrenNode);
                                        
                                        nextXOffset += newGrenNode.width() + 5;
                                    } else {
                                        // Node exists: Update asset texture source, alignment, and visibility
                                        slotNode.image(grenImg);
                                        slotNode.x(nextXOffset);
                                        
                                        if (!slotNode.isVisible()) {
                                            slotNode.show();
                                        }
                                        
                                        nextXOffset += (grenImg?.width || slotNode.width()) + 5;
                                    }
                                } else {
                                    // If the tick data shows this grenade slot is empty, make sure the icon is hidden
                                    const slotNode = t![slotKey] as Konva.Image | undefined;
                                    if (slotNode && slotNode.isVisible()) {
                                        slotNode.hide();
                                    }
                                }
                                })    

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


            
            layerRef.current!.batchDraw()
        }, layerRef.current);
        anim.start()       
        return () => {
            anim.stop()
        };
        }, [isPlaying.playing, round]);

        useEffect(() => {
            if (stats == null) return;
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
                        ...state, vector: place
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
            
        }, [stats, round, size.width, stageDim.height]);

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
                       
                                { playbackRef.current && 
                                    Array.from(playbackRef.current!.player_pos.get(0)!.entries()).map(([playerid, ps], i) => {
                                      const info = playbackRef.current!.player_info.get(playerid)!
                                      const side = info.side
                                      const elements: Partial<PlayerPos> = {};
                                      const color = side == 2 ? "orange" : "blue"
                                      if (side != null){
                                        return (
                                            <Group  key={i} name={"player"} ref={(node) =>{
                                                    if (node != null) {
                                                        playerPosCacheRef.current.set(playerid, elements as PlayerPos)
                                                    }
                                                    return () => {playerPosCacheRef.current.delete(playerid)}
                                                }}>
                                            <Circle
                                                x={ps!.vector.X}
                                                y={ps!.vector.Y}
                                                fill={color}
                                                radius={5}
                                                ref={(node) => { if (node) elements.circle = node }}
                                            />
                                            <Text 
                                                text={info.name} 
                                                x={ps!.vector.X + 5} 
                                                y={ps!.vector.Y - 3} 
                                                fill="white" 
                                                fontSize={10} 
                                                ref={(node) => { if (node) elements.Name = node }}
                                            />
                                        </Group>
                                        )
                                      }
                                    })
                                } 
                        </Group>              
                           {/* These are the player info boxes  */}
                       <Group x={0} >
                            {   playbackRef.current &&
                                Array.from(playbackRef.current!.player_pos.get(0)!.entries()).filter(([playerid]) => {
                                    const info = playbackRef.current!.player_info.get(playerid)!
                                    const side = info.side
                                    return side == 2
                                }).map(([playerid,ps], i) => {
                                    const name = playbackRef.current!.player_info.get(playerid)!.name
                                    const info:PlayerBox = {
                                        width:freeSpace, height:size.height/10, tpad:10, lpad:10, i:i, name:name, ps:ps, playerid:playerid
                                    }
                                    return playerBoxInfo({playerBox:info, hud:hudRef.current, cache:hudCacheRef.current})
                                })
                            }
                        </Group>
                        <Group x={stageDim.width-freeSpace} >
                            {   playbackRef.current &&
                                Array.from(playbackRef.current!.player_pos.get(0)!.entries()).filter(([playerid]) => {
                                    const info = playbackRef.current!.player_info.get(playerid)!
                                    const side = info.side
                                    return side != 2
                                }).map(([playerid, ps], i) => {
                                    const name = playbackRef.current!.player_info.get(playerid)!.name
                                    const info:PlayerBox = {
                                        width:freeSpace, height:size.height/10, tpad:10, lpad:10, i:i, name:name, playerid:playerid, ps:ps
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
                    // TODO
                    }}>Back</button>
                <button onClick={() => {
                        console.log(`BEFORE ANY ACTION TICK:${tickRef.current}`)
                        // setPlaying({...isPlaying, tick_no:tickRef.current, playing: !isPlaying.playing}) 
                        // Going from Playing to Pause
                        if (isPlaying.playing){
                            console.log(`PAUSE TICK:${tickRef.current}`)
                            setPlaying(prev => ({...prev, tick_no:tickRef.current, playing: false}))
                        } else {
                            console.log(`UNPAUSE TICK:${tickRef.current}`)
                            if (progressRef.current) {
                                progressRef.current.value = tickRef.current.toString();
                            }
                            setPlaying(prev => ({...prev, tick_no:tickRef.current, playing: true}))
                        }
                    }}>{ isPlaying.playing == false ? "Play": "Pause"}</button>
                <button onClick={() => {
                    tickRef.current += 500
                    console.log(tickRef.current)
                    }}>forward</button>
            </div>
            <div className="progress">
                {stats && 
                    <input id="playback-slider" onChange={(e) => {
                        const target = parseInt(e.target.value, 10)
                        tickRef.current = target;
                        // setPlaying((prev) => ({ ...prev, tick_no: target }));
                        if (progressRef.current) {
                            progressRef.current.value = target.toString();
                        }
                        const cache:PlayBackCache = {
                            PlayerCache: playerPosCacheRef.current,
                            GrenadeCache: grenadeCache.current,
                            FireCache: fireVertices.current,
                            GrenadeLife: grenadeLife.current,
                            playback: playbackRef.current,
                            Tick: tickRef.current
                        }
                        RedrawAtTicK(cache)
                    }} ref={progressRef} type="range" style={{width: "100%"}}  min={round_begin_ticks.current[0] ?? 0} defaultValue={tickRef.current} max={round_begin_ticks.current[1]}/>
                }
                
            </div>
            <div className="rounds">
                <ul style={{justifyContent:"center", marginTop:"10px"}}>
                {stats && 
                        Array.from({length:stats.rounds}, (_, i) => i+1).map((n, j) => {
                            return <li key={j} onClick={() => {
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