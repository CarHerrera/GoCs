import { Group, Layer, Stage, Text, Rect, Circle, Line, Image } from "react-konva";
import { URLImage } from "./URLImage";
import { useEffect, useState, useRef } from "react";
import Konva from "konva";
import useImage from "use-image";

function playerBoxInfo({width, height, tpad, lpad, i, name}:{width:number, height:number, tpad:number,lpad:number, i:number, name:string}){
    let color ="white"
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
            color="yellow"
        break;
        case 4:
            color="orange"
        break;
        
    }
    const SOURCE = '/equipment/ak47.svg'
    const [nativeImage] = useImage(SOURCE);
    if (i == 0) {
        const bottom = height - 20 - tpad/2;
        return <Group key={i}>
            <Rect width={width} height={height} fill={color} stroke={"black"} strokeWidth={5}/>
            <Rect width={width} height={height *.4} fill={"red"} stroke={"black"} strokeWidth={5}/>
            <Text text={name} fill={"white"} x={lpad}  y={tpad} fontSize={20} ></Text>
            <Text text={"Kills: 0, Assists: 0, Deaths: 0"} fill={"white"} x={lpad}  y={bottom/2 + tpad} fontSize={20} ></Text>
            {/* <Text text={"AK1 GS: HE,F,F,M 30/4"} fill={"white"} x={lpad}  y={bottom} fontSize={20} ></Text>
             */}
             {nativeImage && (
                <Image
                    image={nativeImage}
                    x={lpad}  y={bottom - 10}
                    width={70}
                />
            )}
        </Group>
    } else {
        // This is where the rect starts
        const yStart = i * height + i *tpad;
        const bottom = yStart + height - 20 - tpad/2;
        const mid = yStart + (bottom - yStart)/2 + tpad
        return <Group key={i}>
            
            <Rect width={width} height={height} fill={color} stroke={"black"} y={yStart}strokeWidth={5}/>
            <Rect width={width} height={height *.4} fill={"red"} stroke={"black"} y={yStart} strokeWidth={5}/>
            <Text text={name} fill={"white"} x={lpad}  y={i * height + (i+1) * tpad} fontSize={20} ></Text>
            <Text text={"Kills: 0, Assists: 0, Deaths: 0"} fill={"white"} x={lpad}  y={mid} fontSize={20} ></Text>
            <Text text={`AK${i+1} GS: HE,F,F,M 30/4`} fill={"white"} x={lpad}  y={bottom } fontSize={20} ></Text>
        </Group>
    }
    
}
function TestKonva(){
    
    const player_names = ["Player 1", "Player 2", "Player 3", "Player 4", "Player 5"];
    const map = "de_cache"
    const size = {
        height: 1024,
        width: 1024,
    }
    const playbackDiv = useRef<HTMLDivElement>(null);
    const layerRef = useRef<Konva.Layer>(null);
    const playerRef = useRef<Konva.Group>(null);
    const [getTrueWidth, setTrueWidth] = useState(0);
    const divisor = (getTrueWidth - size.width)/2
    useEffect(() => {
        setTrueWidth(playbackDiv.current?.getBoundingClientRect().width!)
    }, [])
    // const anim = new Konva.Animation(function(frame) {
    //     if (playerRef.current != null){
    //         playerRef.current.rotate((frame.timeDiff * 90) / 100000)
    //     }
    // }, layerRef.current)
    // anim.start()
    const lpad = 10;
    const tpad = 10;
    let id:Number;
    return <div style={{margin:0}}>
                <div>
                    <h1>Team 1 vs Team2</h1>
                </div>
                <div style={{width:"100%"}}>
                    <ul>
                        <li>Fake Tab</li>
                        <li>Fake Tab</li>
                        <li>Fake Tab</li>
                        <li>Fake Tab</li>
                    </ul>
                </div>

                <div style={{
                    display:"flex",
                    flexDirection:"row",
                    width:"100%"
                }}>
                    <div style={{
                        flexGrow:"4",
                        backgroundColor:"#d3d3d3",
                    }}>
                        Left asssssssssssssssssssssss
                        <button onClick={() => {
                            console.log('CLCKED')
                            if (playerRef.current != null){
                                let opacity = 1
                                let c = playerRef.current.findOne((x:Konva.Node)=> {
                                    return x.className == 'Circle'
                                }) as Konva.Circle
                                c.fill("WHITE")
                                
                                
                                
                            }
                        }}>
                            HELLO
                        </button>
                        <button onClick={() => {
                            console.log('CLCKED')
                            if (playerRef.current != null){
                                playerRef.current.getChildren((x) => {
                                    return x.className == 'Circle'
                                }).forEach((c) => {
                                    let t = c as Konva.Circle
                                    t.fill("RED")
                                })
                                
                            }
                        }}>
                            RESET
                        </button>
                    </div>
                    <div ref={playbackDiv} style={{
                        flexGrow:"2",
                        width: "100%"
                    }}>
                        <Stage width={getTrueWidth} height={size.height}>
                            <Layer ref={layerRef} width={getTrueWidth} height={size.height}>
                                <Group x={(getTrueWidth-size.width)/2}>
                                    <URLImage src={`/overviews/${map}.jpg`}  width={size.width} height={size.height}></URLImage> 
                                    <Group ref={playerRef} >
                                        <Circle radius={30}  fill="red" fillPatternY={1} x={(getTrueWidth-size.width)/4} y={size.height/2}></Circle>
                                        <Line points={[(getTrueWidth-size.width)/4, size.height/2, (getTrueWidth-size.width)/4, size.height/2 - 30]} x={(getTrueWidth-size.width)/4} y={size.height/2} stroke={"black"} strokeWidth={5} offsetX={(getTrueWidth-size.width)/4} offsetY={size.height/2}></Line>
                                    </Group>
                                </Group>
                                <Group x={0}>
                                    
                                   {player_names.map((s,i)=> {
                                        return playerBoxInfo({width:(getTrueWidth-size.width)/2, height:size.height/10, tpad:tpad, lpad:lpad, i:i, name:s})
                                   })}
                                
                                                                
                                    
                                </Group>
                                <Group x={getTrueWidth - divisor}>
                                    {
                                        player_names.map((s,i)=> {
                                    // , size.height/10, tpad, lpad, i, 
                                        return playerBoxInfo({width:(getTrueWidth-size.width)/2, height:size.height/10, tpad:tpad, lpad:lpad, i:i, name:s})
                                        })
                                    }
                                </Group>
                            </Layer>
                        </Stage>
                    </div>
                </div>
            </div>
    
}

export default TestKonva;