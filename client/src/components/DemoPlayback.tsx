import { useEffect, useState, useRef } from "react";
import { Layer, Stage, Text, Circle, Group, Rect, Image, Wedge} from 'react-konva';
import { URLImage } from "../URLImage";
import Konva from "konva";
import useImage from "use-image";
import { type_to_svg, WeaponType } from '../helpers/equipIdToSvg'
const PlayerAction = {
    isMoving: 1,
    beginPlanting: 2,
    donePlanting: 3,
    abortedPlant: 4,
} as const;
type PlayerAction = typeof PlayerAction[keyof typeof PlayerAction];
const TrackedEvent = {
	UnknownEvent: 0,
	BombPlanted: 1,
	BombDefused: 2,
	FreezeTimeEnd: 3,
	PlayerKilled: 4,
	FireThrow: 5,
	SmokeThrow: 6,
	FlashThrow: 7,
	HeThrow: 8,
	DecoyThrow: 9,
} as const;
type TrackedEvent = typeof TrackedEvent[keyof typeof TrackedEvent];
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
    smoke_slot: number
    he_slot: number
    fire_slot: number
    decoy_slot: number
    flash_slot1: number
    flash_slot2: number
    view_angle: number
}
interface GrenadeState{
    vector: MapCoordinate;
    grenade: number;
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
    round_events: RoundInfo;
    rounds: number;
    map: {
        pos_x: string,
        pos_y: string,
        scale: string
    }
    teams: Record<string, Record<string, string>>;
}
interface RoundEvent{
    events: TrackedEvent
    player1: string
    player2: string
}
interface RoundInfo {
    // Ticks -> STEAMID -> PLAYER STATE
    player_positions: Record<number, Record<string, PlayerState>>;
    // PLAYERID -> NAME
    player_info: Record<string, PlayerInformation>;
    // TICKS -> UTILID -> GRENADE STATE
    grenade_events: Record<number, Record<string, GrenadeState>>
    // TICKS -> ENTID -> FIRE STATE
    fire_events: Record<number, Record<string, FireState>>
    round_timeline: Record<number, RoundEvent>
}


interface PlaybackState {
    playing: boolean
    round_no: number
    tick_no: number
    ready: boolean
}
interface PlayBackRef{
    // TICK NO -> ROUND_PLAYBACK
    player_pos: Map<number, Map<string, PlayerState>>;
    grenade_pos: Map<number, Map<string, GrenadeState>>;
    fire_vertices: Map<number, Map<string, FireState>>;
    player_info: Map<string, PlayerInformation>;
    round_timeline: Map<number, RoundEvent>;
}

interface EntityInfo{
    kind: number
    status: string
    id: string
    lastTickUpdate: number
    grenade?: Konva.Circle 
    text?: Konva.Text 
    bomb?: Konva.Image 
}
interface EntityState {
    tick: number
    state: string
}
class Entity  {
    grenade: number
    id: string
    changes: EntityState[];
    active: number = 0
    constructor(grenade:number, id:string){
        this.grenade = grenade;
        this.id = id;
        this.changes = [];
    }
    AddState(e:EntityState) {
        this.changes.push(e)
    }
    FirstState() {
        const i = this.changes.length-1
        return i < 0 ? null : this.changes[0]
    } 
    CurrentState() {
        const i = this.changes.length-1
        return i < 0 ? null : this.changes[this.active]
    }
    PreviousState() {
        if (this.active > 0){
            return this.changes[this.active-1]
        }
        return null
    }
    HasNextState() {
        return this.active < this.changes.length-1
    }
    GetNextState(){
        if (this.active < this.changes.length-1){            
            return this.changes[this.active+1]
        } else {
            return null
        }
    }
    SetNextState(){
        if (this.active < this.changes.length-1){
            this.active += 1
        }
    }
    SetPreviousState(){
        if (this.active > 0){
            this.active -= 1
        }
    }
    ResetState(){
        this.active = 0
    }
    LastState(){
        const i = this.changes.length-1
        return i < 0 ? null : this.changes[i]
    }
}

interface FireEntity {
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
    smoke: Konva.Image | null;
    flash1: Konva.Image | null;
    flash2: Konva.Image | null;
    fire: Konva.Image | null;
    he: Konva.Image | null;
    decoy: Konva.Image | null;
    bottomY: number
    fullWidth: number
}
interface PlayerPos {
    circle: Konva.Circle
    Name: Konva.Text
    blndCircle: Konva.Circle
    viewWedge: Konva.Wedge
}
interface FireLifetime{
    lastState: string
    spreadStart: number
    spreadEnd?: number
    vericesStart: MapCoordinate[]
    verticesEnd?: MapCoordinate[]
}
interface PlayBackCache{
    PlayerCache: Map<string, PlayerPos>
    PlayerHudCache: Map<string, PlayerHudCache>
    GrenadeCache: Map<string,EntityInfo>
    FireCache: Map<string, FireEntity>
    GrenadeLife: Map<string, Entity>
    FireLife: Map<string, FireLifetime>
    WeaponImageCache: Map<string, HTMLImageElement>
    playback: PlayBackRef | null
    size: number
    stageWidth: number
    PlaybackGroup: Konva.Group | null;
    HudLayer: Konva.Layer | null;
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
    elements.fullWidth = width
    if (i == 0) {
        const bottom = height - 20 - tpad/2;
        elements.bottomY = bottom - tpad
        
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
                
                <WeaponIcon name="bomb" ref={(n: Konva.Image) => {
                    if (n) {
                        elements.bombImage = n
                    }
                    
                }} src={type_to_svg(404)} x={width - (width/10) - lpad/2}  y={bottom -tpad/2} height={25} width={25}></WeaponIcon>
                <Text name="hp" ref={(n) => {if (n) elements.hpText = n}} text={`${ps.hp}`} fill={"white"} x={width - lpad * 3}  y={tpad} fontSize={12} ></Text>
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${ps.dinero}`} fill={"white"} x={lpad}  y={bottom/2 + tpad} fontSize={12} ></Text>
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}}  src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.smoke = n}} src={type_to_svg(WeaponType.Smokegrenade)} x={width/2}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.flash1 = n}} src={type_to_svg(WeaponType.Flashbang)} x={width/2 + lpad * 1.5 }  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.flash2 = n}} src={type_to_svg(WeaponType.Flashbang)} x={width/2 + lpad * 3.5}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.fire = n}} src={type_to_svg(WeaponType.Molotov)} x={width/2 + lpad *6}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.he = n}} src={type_to_svg(WeaponType.Hegrenade)} x={width/2 + lpad *8}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.decoy = n}} src={type_to_svg(WeaponType.Decoy)} x={width/2 + lpad * 10}  y={bottom -tpad}></WeaponIcon> 
                
                
                
            </Group>
        
    } else {
        // This is where the rect starts
        const yStart = i * height + i *tpad;
        const bottom = yStart + height - 20 - tpad/2;
        const mid = yStart + (bottom - yStart)/2 + tpad
        elements.bottomY = bottom - tpad
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
                <WeaponIcon name="bomb" ref={(n: Konva.Image) => {
                    if (n) {
                        elements.bombImage = n
                    }
                }} src={type_to_svg(404)} x={width - (width/10) - lpad/2}  y={bottom -tpad/2} height={25} width={25}></WeaponIcon>
                <Text width={width} name="name" text={name} fill={"white"} x={lpad}  y={i * height + (i+1) * tpad} fontSize={12} ></Text>
                <Text width={width} ref={(n) => {if (n) elements.statsText = n}} name="stats" text={`Kills: 0, Assists: 0, Deaths: 0 $${ps.dinero}`} fill={"white"} x={lpad}  y={mid} fontSize={12} ></Text>
                <WeaponIcon name="activeWep" ref={(n: Konva.Image) => {if (n) elements.activeWep = n}} src={active_weapon} x={lpad}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.smoke = n}} src={type_to_svg(WeaponType.Smokegrenade)} x={width/2}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.flash1 = n}} src={type_to_svg(WeaponType.Flashbang)} x={width/2 + lpad * 1.5 }  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.flash2 = n}} src={type_to_svg(WeaponType.Flashbang)} x={width/2 + lpad * 3.5}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.fire = n}} src={type_to_svg(WeaponType.Molotov)} x={width/2 + lpad *6}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.he = n}} src={type_to_svg(WeaponType.Hegrenade)} x={width/2 + lpad *8}  y={bottom -tpad}></WeaponIcon>
                <WeaponIcon  ref={(n: Konva.Image) => {if (n) elements.decoy = n}} src={type_to_svg(WeaponType.Decoy)} x={width/2 + lpad * 10}  y={bottom -tpad}></WeaponIcon> 
            </Group>
    }
    
}

function RedrawAtTicK(cache:PlayBackCache, tick:number){
    // Assume we don't need the player positions right now since they are not static or state based. 
    const {PlayerCache, WeaponImageCache, PlayerHudCache, GrenadeCache, FireCache, FireLife, GrenadeLife, playback, size, PlaybackGroup, HudLayer} = cache
    let newGrenade = (id:string, vector:MapCoordinate, radius:number, grenade:string, status:string) : [Konva.Circle, Konva.Text, Konva.Group] => {
        const circle = new Konva.Circle({
            x: vector.X, y:vector.Y, radius: radius, fill: "white"
        })
        const label = new Konva.Text({
            x: vector.X+10, y:vector.Y-3, fontSize:10, fill: "white", text:grenade
        })
        const grenadeProp = new Konva.Group({name:`GRENADE ${grenade} ${status}`, id:id})
        grenadeProp.add(circle, label)
        return [circle, label, grenadeProp]
    }
    if (playback != null){
        GrenadeLife.forEach((ent, id) => {            
            if (GrenadeCache.has(id)){
                const cachedgren = GrenadeCache.get(id)
                if (cachedgren!.grenade) {
                    let tickCorrection = tick
                    while (!playback.grenade_pos.has(tickCorrection)){
                        tickCorrection -= 1
                        if (tickCorrection < ent.FirstState()!.tick) {
                            tickCorrection = ent.FirstState()!.tick
                            break
                        }
                    }
                    let correctedPos:MapCoordinate
                    if (playback.grenade_pos.has(tickCorrection)){
                        if (playback.grenade_pos.get(tickCorrection)!.has(id)){
                            correctedPos = playback.grenade_pos.get(tickCorrection)!.get(id)!.vector
                        } else {
                            if (playback.grenade_pos.has(ent.FirstState()!.tick)){
                                correctedPos = playback.grenade_pos.get(ent.FirstState()!.tick)!.get(id)!.vector
                            } 
                        }
                    }
                    switch (ent.grenade){
                        case WeaponType.Smokegrenade:
                            const next = ent.GetNextState()
                            if(tick > ent.FirstState()!.tick && tick < next!.tick){
                                cachedgren!.grenade!.show()
                                cachedgren!.grenade!.radius(size * 0.01)
                                cachedgren!.grenade!.x(correctedPos!.X)
                                cachedgren!.grenade!.y(correctedPos!.Y)
                                cachedgren!.text!.show()
                                cachedgren!.text!.x(correctedPos!.X)
                                cachedgren!.text!.y(correctedPos!.Y)
                                cachedgren!.status = "FLYING"
                            } else if (tick < ent.LastState()!.tick && tick > ent.GetNextState()!.tick){
                                const state = playback.grenade_pos.get(next!.tick)!.get(id)
                                cachedgren!.grenade.radius(size * 0.035)
                                cachedgren!.grenade!.x(state!.vector.X)
                                cachedgren!.grenade!.y(state!.vector.Y)
                                cachedgren!.grenade!.show()
                                cachedgren!.status = "BLOOMED"  
                            } else if (tick > ent.LastState()!.tick){
                                cachedgren!.grenade!.hide()
                                cachedgren!.text!.hide()
                                cachedgren!.status = "EXPIRED"
                            } else {
                                cachedgren!.grenade!.hide()
                                cachedgren!.text!.hide()
                            }
                        break;
                        case WeaponType.Hegrenade:
                        case WeaponType.Flashbang:
                        case WeaponType.Incgrenade:
                        case WeaponType.Molotov:
                            if (tick < ent.FirstState()!.tick) {
                                cachedgren!.grenade!.hide()
                                cachedgren!.text!.hide()
                                cachedgren!.status = "FLYING"
                            } else if (tick > ent.LastState()!.tick){
                                cachedgren!.grenade!.hide()
                                cachedgren!.text!.hide()
                                if (ent.grenade == WeaponType.Incgrenade || ent.grenade == WeaponType.Molotov){
                                    cachedgren!.status = "LANDED"
                                } else {
                                    cachedgren!.status = "EXPIRED"
                                }
                                
                            } else {
                                cachedgren!.grenade!.show()
                                cachedgren!.text!.show()
                                cachedgren!.grenade!.x(correctedPos!.X)
                                cachedgren!.grenade!.y(correctedPos!.Y)
                                cachedgren!.text!.x(correctedPos!.X)
                                cachedgren!.text!.y(correctedPos!.Y)
                                cachedgren!.status = "FLYING"
                            }
                        break;
                        default:
                        console.warn(`RedrawAtTick: Unknown grenade type ${ent.grenade}`)
                        break;
                    }                    
                } else if (cachedgren!.bomb) {
                    if (tick < ent.FirstState()!.tick) {
                        cachedgren!.bomb!.hide()
                    }
                    if (tick >= ent.LastState()!.tick){
                        const state = playback.grenade_pos.get(ent.LastState()!.tick)!.get(id)!
                        switch (ent.LastState()!.state){
                            case "DEFUSED":
                                cachedgren!.bomb!.x(state.vector.X-25/2)
                                cachedgren!.bomb!.y(state.vector.Y-25/2)
                                cachedgren!.bomb!.hue(240)
                                cachedgren!.bomb!.saturation(100)
                                cachedgren!.bomb!.value(255)
                                cachedgren!.bomb!.show()
                                cachedgren!.bomb!.clearCache()
                                cachedgren!.bomb!.cache()
                                cachedgren!.status = "DEFUSED"
                            break;
                            case "DROPPED":
                                cachedgren!.bomb!.x(state.vector.X-25/2)
                                cachedgren!.bomb!.y(state.vector.Y-25/2)
                                cachedgren!.bomb!.hue(0)
                                cachedgren!.bomb!.saturation(0)
                                cachedgren!.bomb!.value(0)
                                cachedgren!.bomb!.show()
                                cachedgren!.bomb!.clearCache()
                                cachedgren!.bomb!.cache()
                                cachedgren!.status = "DROPPED"
                            break;
                            case "PLANTED":
                                cachedgren!.bomb!.x(state.vector.X-25/2)
                                cachedgren!.bomb!.y(state.vector.Y-25/2)
                                cachedgren!.bomb!.hue(70)
                                cachedgren!.bomb!.saturation(100)
                                cachedgren!.bomb!.value(255)
                                cachedgren!.bomb!.show()
                                cachedgren!.bomb!.clearCache()
                                cachedgren!.bomb!.cache()
                                cachedgren!.status = "PLANTED"
                            break;
                            case "GRABBED":
                                cachedgren!.bomb!.hide()
                                cachedgren!.status = "GRABBED"
                            break;
                        }
                        return
                    }
                    ent.ResetState()
                    while (ent.HasNextState()){
                        const next = ent.GetNextState()
                        if (next != null){  
                            if (tick >= ent.CurrentState()!.tick && tick < next.tick){
                                const state = playback.grenade_pos.get(ent.CurrentState()!.tick)!.get(id)!
                                switch (state.status){
                                    case "PLANTED":
                                        cachedgren!.bomb!.x(state.vector.X-25/2)
                                        cachedgren!.bomb!.y(state.vector.Y-25/2)
                                        cachedgren!.bomb!.hue(70)
                                        cachedgren!.bomb!.saturation(100)
                                        cachedgren!.bomb!.value(255)
                                        cachedgren!.bomb!.show()
                                        cachedgren!.bomb!.clearCache()
                                        cachedgren!.bomb!.cache()
                                        cachedgren!.status = "PLANTED"
                                    break;
                                    case "DEFUSED":
                                        cachedgren!.bomb!.x(state.vector.X-25/2)
                                        cachedgren!.bomb!.y(state.vector.Y-25/2)
                                        cachedgren!.bomb!.hue(240)
                                        cachedgren!.bomb!.saturation(100)
                                        cachedgren!.bomb!.value(255)
                                        cachedgren!.bomb!.show()
                                        cachedgren!.bomb!.clearCache()
                                        cachedgren!.bomb!.cache()
                                        cachedgren!.status = "DEFUSED"
                                    break;
                                    case "DROPPED":
                                        cachedgren!.bomb!.x(state.vector.X-25/2)
                                        cachedgren!.bomb!.y(state.vector.Y-25/2)
                                        cachedgren!.bomb!.hue(0)
                                        cachedgren!.bomb!.saturation(0)
                                        cachedgren!.bomb!.value(0)
                                        cachedgren!.bomb!.show()
                                        cachedgren!.bomb!.clearCache()
                                        cachedgren!.bomb!.cache()
                                        cachedgren!.status = "DROPPED"
                                    break;
                                    case "GRABBED":
                                        cachedgren!.bomb!.hide()
                                        cachedgren!.status = "GRABBED"
                                    break;
                                    default:
                                        console.log(`RedrawAtTick: Unknown bomb state ${state.status}`)
                                    break
                                }
                            }
                        }
                        ent.SetNextState()
                    }
                }
            } else {
                if (!PlaybackGroup) { 
                    console.warn('RedrawAtTick: PlaybackGroup is null, cannot create objects') 
                    return 
                }
                // No need to create object yet
                
                if (tick < ent.FirstState()!.tick) {return}
                switch(ent.grenade){
                    case WeaponType.Smokegrenade:
                        // WILL HAVE FLYING -> BLOOM -> EXPIRED
                        const next = ent.GetNextState()!
                        //  firstStateTick < tick < nextStateTick which means it is flying
                        let tickCorrection = tick                    
                        if (tick > ent.FirstState()!.tick && tick < next.tick){
                            while (!playback.grenade_pos.has(tickCorrection)){
                                tickCorrection -= 1
                                if (tickCorrection < ent.FirstState()!.tick) {
                                    tickCorrection = ent.FirstState()!.tick
                                    break
                                }
                                
                            }
                            const state = playback.grenade_pos.get(tickCorrection)!.get(id)
                            const [circle, label, grenadeProp] = newGrenade(id, state!.vector, size * 0.035, "SMOKE", state!.status)
                            let cache:EntityInfo = { kind: state!.grenade, status: state!.status, id: id, lastTickUpdate:ent.FirstState()!.tick, grenade: circle, text: label }
                            GrenadeCache.set(id, cache)
                            PlaybackGroup!.add(grenadeProp)
                        } else if (tick >= next.tick && tick <= ent.LastState()!.tick) {
                            const state = playback.grenade_pos.get(next.tick)!.get(id)
                            const [circle, label, grenadeProp] = newGrenade(id, state!.vector, size * 0.01, "SMOKE", state!.status)                                
                            let cache:EntityInfo = { kind: state!.grenade, status: state!.status, id: id, lastTickUpdate:tick, grenade: circle, text: label }
                            GrenadeCache.set(id, cache)
                            PlaybackGroup!.add(grenadeProp)
                        }
                        
                    break;
                    case WeaponType.Incgrenade:
                    case WeaponType.Molotov:
                    case WeaponType.Hegrenade:
                    case WeaponType.Flashbang:
                        if (tick > ent.FirstState()!.tick && tick < ent.LastState()!.tick){
                            let grenade = ""
                                if (ent.grenade == WeaponType.Hegrenade){
                                    grenade = "HE"
                                } else if (ent.grenade == WeaponType.Flashbang) {
                                    grenade = "Flash"
                                } else {
                                    grenade = "FIRE"
                                }
                                let tickCorrection = tick
                                while (!playback.grenade_pos.has(tickCorrection)){
                                    tickCorrection -= 1
                                }
                                const state = playback.grenade_pos.get(ent.FirstState()!.tick)!.get(id)
                                const [circle, label, grenadeProp] = newGrenade(id, state!.vector, size * 0.01, grenade, state!.status)
                                let cache:EntityInfo = { kind: state!.grenade, status: state!.status, id: id, lastTickUpdate:ent.FirstState()!.tick, grenade: circle, text: label }
                                GrenadeCache.set(id, cache)
                                PlaybackGroup!.add(grenadeProp)
                        }
                    break;
                    case WeaponType.C4:
                        while(ent.HasNextState()){
                            const next = ent.GetNextState()
                            if (next != null){
                                if (tick > ent.CurrentState()!.tick && tick < next.tick){
                                    const state = playback.grenade_pos.get(ent.CurrentState()!.tick)!.get(id)!
                                    let bomb = new Konva.Group({name:`BOMB ${state.status}`, id:"-1"})
                                    const bombSvg = WeaponImageCache.get(type_to_svg(404))
                                    let rect = new Konva.Image({
                                        filters:[Konva.Filters.HSV],  image:bombSvg, x: state.vector.X-25/2, y: state.vector.Y-25/2, width:25, height:25,
                                    })
                                    rect.cache()
                                    let cache:EntityInfo = {
                                        kind: WeaponType.C4, status: state.status, id: id, lastTickUpdate:ent.CurrentState()!.tick, bomb:rect
                                    }
                                    switch (state.status) {
                                        case "DROPPED":
                                            bomb.add(rect)
                                            bomb.show()
                                            rect.clearCache()
                                            rect.cache()
                                            GrenadeCache.set(id, cache)
                                            PlaybackGroup!.add(bomb)
                                        break;
                                        case "PLANTED":
                                            rect.hue(70)
                                            rect.saturation(100)
                                            rect.value(255)
                                            bomb.show()
                                            bomb.add(rect)
                                            rect.clearCache()
                                            rect.cache()
                                            GrenadeCache.set(id, cache)
                                            PlaybackGroup!.add(bomb)
                                        break;
                                        case "GRABBED":
                                            return
                                        default:
                                            console.log(`NEW ${state.grenade} ${state.status} has not been handled yet.`)
                                        return;
                                    }
                                }
                                ent.SetNextState()
                            } 
                        }
                        if (tick > ent.LastState()!.tick){
                            console.log(`BOMB STATE ${ent.LastState()!.state}`, ent)
                            const state = playback.grenade_pos.get(ent.LastState()!.tick)!.get(id)!
                            let bomb = new Konva.Group({name:`BOMB ${state.status}`, id:"-1"})
                            const bombSvg = WeaponImageCache.get(type_to_svg(404))
                            let rect = new Konva.Image({
                                filters:[Konva.Filters.HSV],  image:bombSvg, x: state.vector.X-25/2, y: state.vector.Y-25/2, width:25, height:25,
                            })
                            rect.cache()
                            let cache:EntityInfo = {
                                kind: WeaponType.C4, status: state.status, id: id, lastTickUpdate:ent.LastState()!.tick, bomb:rect
                            }

                            switch (state.status) {
                                case "DROPPED":
                                    bomb.add(rect)
                                    bomb.show()
                                    GrenadeCache.set(id, cache)
                                    PlaybackGroup!.add(bomb)
                                break;

                            }
                            ent.ResetState()
                        }
                    break;
                    default:
                        console.warn(`RedrawAtTick: Unknown grenade type ${ent.grenade}`)
                    break;
                }
            }
        })
        
        FireLife.forEach((fl, id) => {
            if (FireCache.has(id)){
                const fireEnt = FireCache.get(id)
                // console.log(`Fireent ${id} has already been created at ${tick} ${fl.spreadStart}`)
                if (fireEnt) {
                    if (tick < fl.spreadStart) {
                        fireEnt.circle!.hide()
                        fireEnt.spread!.hide()
                        fireEnt.state = "STARTING"
                        fireEnt.vertices = 1
                        const mapCoordinate:MapCoordinate = {
                            X: fl.vericesStart[0].X,
                            Y: fl.vericesStart[0].Y
                        }
                        fireEnt.spread!.setAttr("customVertices", [mapCoordinate])
                    } else if (tick > fl.spreadEnd!){
                        fireEnt.circle!.hide()
                        fireEnt.spread!.hide()
                        fireEnt.state = "ENDING"
                        fireEnt.vertices = fl.verticesEnd!.length
                        fireEnt.spread!.setAttr("customVertices", [0,0])
                    } else {
                        console.log("FIRE SHOULD EXISTS")
                        fireEnt.circle!.hide()
                        fireEnt.spread!.show()
                        fireEnt.state = "SPREADING"
                        fireEnt.vertices = fl.verticesEnd!.length
                        fireEnt.spread!.setAttr("customVertices", fl.verticesEnd!)
                    }
                }
            } else {
                if (!PlaybackGroup) { 
                    console.warn('RedrawAtTick: PlaybackGroup is null, cannot create objects') 
                    return 
                }
                if (tick > fl.spreadStart && tick < fl.spreadEnd!){
                    
                    const state = playback.fire_vertices.get(fl.spreadStart)!.get(id)!
                    let fire = new Konva.Group({name:`FIRE ${state.status}`})
                    let circl = new Konva.Circle({
                        x : state.vertices[0].X , y: state.vertices[0].Y, radius: size * .01, fill:"orange"
                    })
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
                    fire.add(circl, spread)
                    const fireEnt: FireEntity = {
                        circle: circl, state: "SPREADING", vertices: fl.vericesStart.length, spread:spread, 
                    }
                    circl.hide()
                    FireCache.set(id, fireEnt)
                    PlaybackGroup!.add(fire)
                }
            }
        })
        
        PlayerCache.forEach((pp, id) => {
            let corrTick = tick
            while (!playback.player_pos.has(corrTick)){
                corrTick -= 1
            }
            const player = playback.player_pos.get(corrTick)!.get(id)
            if (player){
                const vect = player.vector
                pp.Name!.x(vect.X)
                pp.Name!.y(vect.Y)
                pp.circle!.x(vect.X)
                pp.circle!.y(vect.Y)
                pp.blndCircle!.x(vect.X)
                pp.blndCircle!.y(vect.Y)
                pp.blndCircle!.opacity(player.blind_dur/4.5)
                pp.viewWedge!.x(vect.X)
                pp.viewWedge!.y(vect.Y)
                pp.viewWedge!.rotation(-player.view_angle - 45)
                const hud = PlayerHudCache.get(id)
                hud!.statsText.text(`Kills: ${player.kills}, Assists: ${player.assists}, Deaths: ${player.deaths} $${player.dinero}`)
                hud!.hpBar.width(player.hp/100 * (hud!.fullWidth))
                hud!.hpText.text(`${player.hp}`)
                const src = type_to_svg(player!.active_weapon)
                const cacheImg = WeaponImageCache.get(src)
                if (hud!.activeWep){
                    if (cacheImg && hud!.activeWep.image() !== cacheImg){
                        hud!.activeWep.image(cacheImg)
                    }
                }
                let start = hud!.fullWidth/2
                if (player.smoke_slot !== 0){
                    hud!.smoke!.x(start)
                    hud!.smoke!.show()
                    hud!.smoke!.clearCache()
                    hud!.smoke!.cache()
                    start +=  hud!.smoke!.getWidth() + 3
                } else {
                    if (hud!.smoke && hud!.smoke.isVisible()){
                        hud!.smoke.hide()
                    }
                }

                if (player.flash_slot1 !== 0){
                    hud!.flash1!.x(start)
                    hud!.flash1!.show()
                    hud!.flash1!.clearCache()
                    hud!.flash1!.cache()
                    start += hud!.flash1!.getWidth() 
                } else {
                    if (hud!.flash1 && hud!.flash1.isVisible()){
                        hud!.flash1.hide()
                    }
                
                }

                if (player.flash_slot2 !== 0){
                    hud!.flash2!.x(start)
                    hud!.flash2!.show()
                    hud!.flash2!.clearCache()
                    hud!.flash2!.cache()
                    start += hud!.flash2!.getWidth() 
                } else {
                    if (hud!.flash2 && hud!.flash2.isVisible()){
                        hud!.flash2.hide()
                    }
                }

                if (player.fire_slot !== 0){
                    hud!.fire!.x(start)
                    hud!.fire!.show()
                    hud!.fire!.clearCache()
                    hud!.fire!.cache()
                    start += hud!.fire!.getWidth() 
                } else {
                    if (hud!.fire && hud!.fire.isVisible()){
                        hud!.fire.hide()
                    }
                }

                if (player.decoy_slot !== 0){
                    hud!.decoy!.x(start)
                    hud!.decoy!.show()
                    hud!.decoy!.clearCache()
                    hud!.decoy!.cache()
                    start += hud!.decoy!.getWidth() 
                } else {
                    if (hud!.decoy && hud!.decoy.isVisible()){
                        hud!.decoy.hide()
                    }
                }

                if (player.he_slot !== 0){
                    hud!.he!.x(start)
                    hud!.he!.show()
                    hud!.he!.clearCache()
                    hud!.he!.cache()
                    start += hud!.he!.getWidth()
                } else {
                    if (hud!.he && hud!.he.isVisible()){
                        hud!.he.hide()
                    }
                }
                if (player.hasBomb){
                    hud!.bombImage!.show()
                    hud!.bombImage!.clearCache()  
                    
                    hud!.bombImage!.cache() 
                } else {
                    hud!.bombImage!.hide()
                }
                
            } else {
                const hud = PlayerHudCache.get(id)
                pp.Name!.x(-1000)
                pp.Name!.y(-1000)
                pp.circle!.x(-1000)
                pp.circle!.y(-1000)
                pp.blndCircle!.x(-1000)
                pp.blndCircle!.y(-1000)
                pp.viewWedge!.x(-1000)
                pp.viewWedge!.y(-1000)                
                hud!.hpBar.width(0)
                hud!.hpText.text(`0`)
            }
            HudLayer!.batchDraw()
        })
        
    } else {
        console.warn('RedrawAtTick: playback is null, returning early')
        return
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
        const [isPlaying, setPlaying] = useState<PlaybackState>({playing: false, round_no:1, tick_no: 0, ready: false});
        const playbackContainer = useRef<HTMLDivElement>(null);
        const round_begin_ticks = useRef<number[]>([]);
        const weaponImageCacheRef = useRef<Map<string, HTMLImageElement>>(new Map());
        const playerPosCacheRef = useRef<Map<string, PlayerPos>>(new Map())
        const grenadeCache = useRef<Map<string, EntityInfo>>(new Map());
        const grenadeLife = useRef<Map<string, Entity>>(new Map());
        const fireLife = useRef<Map<string, FireLifetime>>(new Map());
        const fireVertices = useRef<Map<string, FireEntity>>(new Map());
        const tickRef = useRef(isPlaying.tick_no);
        const hudRef = useRef<Map<string, Konva.Group>>(new Map());
        const hudCacheRef = useRef<Map<string, PlayerHudCache>>(new Map());
        
        const fullPlaybacRef = useRef<PlayBackCache>(null)
        const progressRef = useRef<HTMLInputElement>(null);
        // ROUND NO -> TICK NO -> PLAYBACK REF
        const playbackRef = useRef<PlayBackRef>(null);
        const videoPlaybackLayer = useRef<Konva.Layer>(null)
        const hudLayerRef = useRef<Konva.Layer>(null);
        const videoGroup = useRef<Konva.Group>(null);
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
        // Animation
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
                            cache.viewWedge.x(ps!.vector.X)
                            cache.viewWedge.y(ps!.vector.Y)
                            cache.viewWedge.rotation(-ps!.view_angle - 45)
                            if (ps.blind_dur > 0){
                                cache.blndCircle.x(ps!.vector.X)
                                cache.blndCircle.y(ps!.vector.Y)
                                cache.blndCircle.opacity(ps.blind_dur/5)
                            } else {
                                cache.blndCircle.opacity(0)
                            }
                        } else {
                            cache.circle.x(-1000)
                            cache.circle.y(-1000)
                            cache.blndCircle.x(-1000)
                            cache.blndCircle.y(-1000)
                            cache.viewWedge.x(-1000)
                            cache.viewWedge.y(-1000)
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
                            if (gren!.kind != WeaponType.C4){
                                if (gren!.status == state.status){
                                    switch (state.status){
                                        case "FLYING":
                                            gren!.grenade!.x(state.vector.X) 
                                            gren!.grenade!.y(state.vector.Y) 
                                            gren!.text!.x(state.vector.X + 5)
                                            gren!.text!.y(state.vector.Y - 3)
                                            gren!.grenade!.radius(size.height * .01)
                                            if (!gren!.grenade!.isVisible()){
                                                gren!.grenade!.show()
                                                gren!.text!.show()
                                            }
                                            gren!.status = "FLYING"
                                        break;
                                        case "BLOOMED":
                                            gren!.grenade!.radius(size.height * .035)
                                            gren!.status = "BLOOMED"
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
                            } else {
                                if (gren!.status != state.status){
                                    switch (state.status){
                                        case "GRABBED":
                                            gren!.bomb!.hide()
                                        break;
                                        case "DEFUSED":
                                            gren!.bomb!.hue(240)
                                            gren!.bomb!.saturation(100)
                                            gren!.bomb!.value(255)
                                            gren!.bomb!.clearCache()
                                            gren!.bomb!.cache()
                                            gren!.bomb!.show()
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
                            const mainGroup = videoPlaybackLayer.current!.findOne("#mainPlayer") as Konva.Group

                            if (state.grenade == WeaponType.C4) {
                                // console.log("BOMB NEEDS TO BE ADDED TO MAP")
                                let bomb = new Konva.Group({name:`BOMB ${state.status}`, id:id})
                                let rect = new Konva.Image({
                                      filters:[Konva.Filters.HSV],  image:bombSvg, x: state.vector.X-25/2, y: state.vector.Y-25/2, width:25, height:25,
                                })
                                rect.cache()
                                let cache:EntityInfo = {
                                    kind: WeaponType.C4, status: state.status, id: id, lastTickUpdate:tickRef.current, bomb:rect
                                }
                                grenadeCache.current.set(id, cache)
                                switch (state.status) {
                                    case "DROPPED":
                                        bomb.add(rect)
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
                                let grenade = ""
                                switch(state.grenade){
                                    case WeaponType.Smokegrenade:
                                        grenade = "SMOKE"
                                    break;
                                    case WeaponType.Hegrenade:
                                        grenade = "HE"
                                    break;
                                    case WeaponType.Flashbang:
                                        grenade = "Flash"
                                    break;
                                    case WeaponType.Incgrenade:
                                        grenade = "Incendiary"
                                    break;
                                    case WeaponType.Molotov:
                                        grenade = "Molly"
                                    break;
                                }
                                let gren = new Konva.Group({name:`GRENADE ${grenade} ${state.status}`, id:id})
                                let circl = new Konva.Circle({
                                        x: state.vector.X, y: state.vector.Y, radius: size.width * .01 , fill:"white"
                                    })
                                let label = new Konva.Text({
                                        x: state.vector.X + 5, y: state.vector.Y - 3, text: grenade,
                                        fill:"white" , fontSize:10 
                                })
                                let cache:EntityInfo = {
                                    kind: state.grenade, status: state.status, id: id, lastTickUpdate:tickRef.current, grenade: circl, text: label
                                }
                                grenadeCache.current.set(id, cache)
                                gren.add(circl, label)
                                
                                mainGroup.add(gren)
                            }
                            
                        }
                    })
                }

                if(playbackRef.current!.fire_vertices.has(tickRef.current)){
                    const mainGroup = videoPlaybackLayer.current!.findOne("#mainPlayer") as Konva.Group
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
                                            fireInfo.spread!.show()
                                        }
                                        if (fireInfo.circle != null && fireInfo.circle!.isVisible()){
                                                fireInfo.circle!.hide()
                                        }
                                    break;
                                    case "STARTING":
                                        console.log("SHOULD BE STARTING")
                                        if (fireInfo.spread){
                                            fireInfo.spread!.show()
                                            fireInfo.circle!.show()
                                            fireInfo.vertices = 1
                                        } else {
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
                                            fireInfo.circle!.hide()
                                        }
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
                                        if (fireInfo.spread){
                                            fireInfo.spread!.show()
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
                                x : state.vertices[0].X , y: state.vertices[0].Y, radius: size.height * .01, fill:"orange"
                            })
                            const mainGroup = videoPlaybackLayer.current!.findOne("#mainPlayer") as Konva.Group
                            fire.add(circl)
                            const fireEnt: FireEntity = {
                                circle: circl, state: "STARTING", vertices: 1
                            }
                            fireVertices.current.set(id, fireEnt)
                            mainGroup.add(fire)
                        }
                    })
                }
                
            }

            if (hudRef.current != null) {
                    // console.log("IN HUD")
                    // console.log(hudRef.current)
                hudRef.current!.forEach((g, id) => {
                    if (playbackRef.current!.player_pos.has(tickRef.current)){
                        const p = playbackRef.current!.player_pos.get(tickRef.current)?.get(id)
                        if (p != null){
                            let t = hudCacheRef.current.get(id)
                            let y = t!.bottomY
                            t!.statsText.text(`Kills: ${p.kills}, Assists: ${p.assists}, Deaths: ${p.deaths} $${p.dinero}`)
                            t!.hpBar.width(p.hp/100 * (stageDim.width-size.width)/2)
                            t!.hpText.text(`${p.hp}`)
                            // const grenades = [p.slot1, p.slot2, p.slot3, p.slot4]
                            const src = type_to_svg(p.active_weapon)
                            const cacheImg = weaponImageCacheRef.current.get(src)
                            if (t!.activeWep){
                                if (cacheImg && t!.activeWep.image() !== cacheImg){
                                    t!.activeWep.image(cacheImg)
                                }                               
                            }
                            let start = (stageDim.width-size.width)/2
                            if (p.smoke_slot !== 0){
                                t!.smoke!.x(start)
                                t!.smoke!.show()
                                t!.smoke!.clearCache()
                                t!.smoke!.cache()
                                start +=  t!.smoke!.getWidth() + 3
                            } else {
                                if (t!.smoke && t!.smoke.isVisible()){
                                    t!.smoke.hide()
                                }
                            }

                            if (p.flash_slot1 !== 0){
                                t!.flash1!.x(start)
                                t!.flash1!.show()
                                t!.flash1!.clearCache()
                                t!.flash1!.cache()
                                start += t!.flash1!.getWidth() 
                            } else {
                                if (t!.flash1 && t!.flash1.isVisible()){
                                    t!.flash1.hide()
                                }
                            
                            }

                            if (p.flash_slot2 !== 0){
                                t!.flash2!.x(start)
                                t!.flash2!.show()
                                t!.flash2!.clearCache()
                                t!.flash2!.cache()
                                start += t!.flash2!.getWidth() 
                            } else {
                                if (t!.flash2 && t!.flash2.isVisible()){
                                    t!.flash2.hide()
                                }
                            }

                            if (p.fire_slot !== 0){
                                t!.fire!.x(start)
                                t!.fire!.show()
                                t!.fire!.clearCache()
                                t!.fire!.cache()
                                start += t!.fire!.getWidth() 
                            } else {
                                if (t!.fire && t!.fire.isVisible()){
                                    t!.fire.hide()
                                }
                            }

                            if (p.decoy_slot !== 0){
                                t!.decoy!.x(start)
                                t!.decoy!.show()
                                t!.decoy!.clearCache()
                                t!.decoy!.cache()
                                start += t!.decoy!.getWidth() 
                            } else {
                                if (t!.decoy && t!.decoy.isVisible()){
                                    t!.decoy.hide()
                                }
                            }

                            if (p.he_slot !== 0){
                                t!.he!.x(start)
                                t!.he!.show()
                                t!.he!.clearCache()
                                t!.he!.cache()
                                start += t!.he!.getWidth()
                            } else {
                                if (t!.he && t!.he.isVisible()){
                                    t!.he.hide()
                                }
                            }
                            if (p.hasBomb){
                                t!.bombImage!.show()
                                t!.bombImage!.clearCache()  
                                t!.bombImage!.cache() 
                            } else {
                                t!.bombImage!.hide()
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
            hudLayerRef.current?.batchDraw()
            videoPlaybackLayer.current?.batchDraw()
        });
        anim.start()       
        return () => {
            anim.stop()
        };
        }, [isPlaying.playing, round]);
        // Init
        useEffect(() => {
            if (stats == null) return;
            if (progressRef.current) {
                progressRef.current.value = tickRef.current.toString();
            }
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
            const timeline = Array.from(Object.entries(stats.round_events.round_timeline))
            let tick_map:PlayBackRef = {
                player_pos: new Map<number, Map<string, PlayerState>>(),
                grenade_pos: new Map<number, Map<string, GrenadeState>>(),
                fire_vertices: new Map<number, Map<string, FireState>>(),
                player_info: new Map<string, PlayerInformation>(),
                round_timeline: new Map<number, RoundEvent>()
            };
            grenadeLife.current = new Map();
            fireLife.current = new Map();
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
                    const stateNew = grenadeLife.current.get(grenid)
                    if (stateNew == null){
                        const ent = new Entity(grenstate.grenade, grenid)
                        ent.AddState({tick:Number(tick), state:grenstate.status})
                        grenadeLife.current.set(grenid, ent)
                    } else {
                        if (stateNew.LastState()){     
                            if (stateNew.LastState()!.state != grenstate.status){
                                stateNew.AddState({tick:Number(tick), state:grenstate.status})
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
                     const fire_life = fireLife.current.get(entid)
                    if (fire_life == null){
                       const ent:FireLifetime = {
                            spreadStart: parseInt(tick),
                            vericesStart: vertices,
                            lastState: "STARTING"
                       } 
                       fireLife.current.set(entid, ent)
                    } else {
                        if (state.status != fire_life.lastState){
                            switch(state.status){
                                case "SPREADING":
                                    fire_life.vericesStart = vertices
                                    fire_life.lastState = "SPREADING"
                                break;
                                case "ENDING":
                                    fire_life.spreadEnd = parseInt(tick)
                                    fire_life.verticesEnd = vertices
                                    fire_life.lastState = "ENDING"
                                break;
                            }
                            fireLife.current.set(entid, fire_life)
                        }
                    }
                    fire.set(entid, fire_state)
                })
                tick_map.fire_vertices.set(Number(tick), fire)
            })
            timeline.forEach(([tick, re]) => {
                if (Number(tick) < round_begin_ticks.current[0]) {
                    return
                }
                tick_map.round_timeline.set(Number(tick),re)
            })
            playbackRef.current = tick_map
            // console.log(playbackRef.current)
            const player_info = Array.from(Object.entries(stats.round_events.player_info))
            player_info.forEach(([playerid, playername]) => {
                playbackRef.current!.player_info.set(playerid, playername)
            })
            
            const cache:PlayBackCache = {
                PlayerCache: playerPosCacheRef.current,
                PlayerHudCache: hudCacheRef.current,
                GrenadeCache: grenadeCache.current,
                FireCache: fireVertices.current,
                GrenadeLife: grenadeLife.current,
                playback: playbackRef.current,
                FireLife: fireLife.current,
                WeaponImageCache: weaponImageCacheRef.current,
                HudLayer: hudLayerRef.current,
                PlaybackGroup: videoGroup.current,
                size: size.height,
                stageWidth: stageDim.width
            }
            fullPlaybacRef.current = cache
            setPlaying({...isPlaying, ready:true})
        }, [stats, round, size.width, stageDim.height]);
        // console.log(grenadeCacheNew.current)
        const freeSpace:number = (stageDim.width-size.width)/2
    return <>
        <div id="playbackGrid" >
            
            <div className="options">
                <ul style={{border:"solid"}}>
                    <li>Round Events</li>
                    <li>Notes</li>
                    <li>Drawing mode</li>
                    <li>Settings</li>
                </ul>
            <div>
                    <ol>
                        {
                            isPlaying.ready && 
                                Array.from(playbackRef.current!.round_timeline.entries()).map(([tick, re], i)=>{
                                    const player1 = playbackRef.current!.player_info.get(re.player1)
                                    const player2 = playbackRef.current!.player_info.get(re.player2)
                                    let msg = ""
                                    switch(re.events){
                                        case TrackedEvent.FreezeTimeEnd:
                                            msg += `Freeze time ended at ${tick}`
                                            break
                                        case TrackedEvent.PlayerKilled:
                                            msg += `${player1?.name} killed ${player2?.name} at ${tick}`
                                            break
                                        case TrackedEvent.BombPlanted:
                                            msg += `${player1?.name} Planted the bomb at ${tick}`
                                            break
                                        case TrackedEvent.BombDefused:
                                            msg += `${player1?.name} defused the bomb at ${tick}`
                                            break
                                        case TrackedEvent.SmokeThrow:
                                            msg += `${player1?.name} threw a smoke ${tick}`
                                            break
                                        case TrackedEvent.FlashThrow:
                                            msg += `${player1?.name} threw a flash ${tick}`
                                            break
                                        case TrackedEvent.HeThrow:
                                            msg += `${player1?.name} threw an HE ${tick}`
                                            break
                                        case TrackedEvent.FireThrow:
                                            msg += `${player1?.name} threw a fire grenade ${tick}`
                                            break
                                        default:
                                            msg += `Nothing`
                                            break
                                    }
                                    return <li key={i}>{msg}</li>
                                })
                        }
                    </ol>
                </div>
            </div>
            <div id="playbackMap" ref={playbackContainer}>
                <Stage  width={stageDim.width} height={stageDim.height}>
                    <Layer ref={videoPlaybackLayer}   >
                        <Group x={freeSpace} id={"mainPlayer"} ref={videoGroup}>
                            <URLImage src={`/overviews/${map}.jpg`} name="map"  width={size.width} height={stageDim.height}></URLImage>     
                       
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
                                            <Circle
                                                x={ps!.vector.X}
                                                y={ps!.vector.Y}
                                                fill={"white"}
                                                radius={5}
                                                opacity={0}
                                                ref={(node) => { if (node) elements.blndCircle = node }}
                                            />
                                            <Wedge
                                                x={ps!.vector.X}
                                                y={ps!.vector.Y}
                                                angle={90} 
                                                fill={"gray"} 
                                                radius={5} 
                                                rotation={-ps!.view_angle - 45} 
                                                opacity={100}
                                                ref={(node) => {if (node) elements.viewWedge = node}}
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
                    </Layer>
                    <Layer ref={hudLayerRef}>
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
                    RedrawAtTicK(fullPlaybacRef.current!, tickRef.current)
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
                        RedrawAtTicK(fullPlaybacRef.current!, tickRef.current)
                    }}>{ isPlaying.playing == false ? "Play": "Pause"}</button>
                <button onClick={() => {
                    tickRef.current += 500
                    console.log(tickRef.current)
                    }}>forward</button>
            </div>
            <div className="progress">
                {playbackRef.current && 
                    <input id="playback-slider" onChange={(e) => {
                        const target = parseInt(e.target.value, 10)
                        tickRef.current = target;
                        setPlaying((prev) => ({ ...prev, tick_no: target, playing:false}));
                        if (progressRef.current) {
                            progressRef.current.value = target.toString();
                        }
                        
                        RedrawAtTicK(fullPlaybacRef.current!, tickRef.current)
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
                                        node.hide()
                                    }

                                });
                                fireLife.current.forEach((_, id) => {
                                    const ent = fireVertices.current.get(id)
                                    if (ent){
                                        if (ent.circle){
                                            ent.circle.destroy()
                                        }
                                        if(ent.spread){
                                            ent.spread.destroy()
                                        }
                                    }
                                })
                                grenadeLife.current.forEach((_, id) => {
                                    const gren = grenadeCache.current.get(id)
                                    if (gren){
                                        if (gren.bomb){
                                            gren.bomb.remove()
                                        }
                                        if (gren.grenade){
                                            gren.grenade.destroy()
                                        }
                                        if (gren.text){
                                            gren.text.destroy()
                                        }
                                    }
                                })
                                grenadeLife.current.clear()
                                fireLife.current.clear()
                                grenadeCache.current.clear()
                                fireVertices.current.clear()
                                setPlaying({tick_no:0, playing:false, round_no: n, ready:false})
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