// Whiteboard.tsx
import { useState, useRef } from 'react';
import { Stage, Layer, Line, Rect, Circle, Image as KonvaImage } from 'react-konva';
import useImage from 'use-image';

type Tool = 'pen' | 'rect' | 'circle' | 'eraser';

interface DrawLine { tool: Tool; points: number[]; color: string; }
interface Shape { type: 'rect' | 'circle'; x: number; y: number; w: number; h: number; color: string; }

const MAPS = ['de_mirage', 'de_inferno', 'de_ancient', 'de_nuke', 'de_dust2', 'de_overpass', 'de_anubis'];

const Whiteboard: React.FC = () => {
  const [tool, setTool] = useState<Tool>('pen');
  const [color, setColor] = useState('#ff3b3b');
  const [selectedMap, setSelectedMap] = useState('de_mirage');
  const [lines, setLines] = useState<DrawLine[]>([]);
  const [shapes, setShapes] = useState<Shape[]>([]);
  const isDrawing = useRef(false);
  const stageRef = useRef<any>(null);
  const [mapImage] = useImage(`/overviews/${selectedMap}.jpg`);

  const handleMouseDown = (e: any) => {
    isDrawing.current = true;
    const pos = e.target.getStage().getPointerPosition();
    if (tool === 'pen' || tool === 'eraser') {
      setLines([...lines, { tool, points: [pos.x, pos.y], color: tool === 'eraser' ? '#1a1a2e' : color }]);
    } else {
      setShapes([...shapes, { type: tool, x: pos.x, y: pos.y, w: 0, h: 0, color }]);
    }
  };

  const handleMouseMove = (e: any) => {
    if (!isDrawing.current) return;
    const pos = e.target.getStage().getPointerPosition();
    if (tool === 'pen' || tool === 'eraser') {
      const last = lines[lines.length - 1];
      last.points = last.points.concat([pos.x, pos.y]);
      setLines([...lines.slice(0, -1), last]);
    } else {
      const last = shapes[shapes.length - 1];
      last.w = pos.x - last.x;
      last.h = pos.y - last.y;
      setShapes([...shapes.slice(0, -1), last]);
    }
  };

  const handleMouseUp = () => { isDrawing.current = false; };

  const handleExport = () => {
    const uri = stageRef.current.toDataURL({ pixelRatio: 2 });
    const link = document.createElement('a');
    link.download = `${selectedMap}-strat.png`;
    link.href = uri;
    link.click();
  };

  const clearBoard = () => { setLines([]); setShapes([]); };

  return (
    <div className="whiteboard">
      <div className="whiteboard-toolbar">
        <select value={selectedMap} onChange={e => setSelectedMap(e.target.value)}>
          {MAPS.map(m => <option key={m} value={m}>{m}</option>)}
        </select>
        <button className={tool === 'pen' ? 'active' : ''} onClick={() => setTool('pen')}>✏️ Pen</button>
        <button className={tool === 'rect' ? 'active' : ''} onClick={() => setTool('rect')}>▭ Box</button>
        <button className={tool === 'circle' ? 'active' : ''} onClick={() => setTool('circle')}>◯ Circle</button>
        <button className={tool === 'eraser' ? 'active' : ''} onClick={() => setTool('eraser')}>Eraser</button>
        <input type="color" value={color} onChange={e => setColor(e.target.value)} />
        <button onClick={clearBoard}>Clear</button>
        <button className="export-btn" onClick={handleExport}>Export Image</button>
      </div>

      <Stage
        ref={stageRef}
        width={1024} height={1024}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        className="whiteboard-stage"
      >
        <Layer>
          {mapImage && <KonvaImage image={mapImage} width={1024} height={1024} />}
        </Layer>
        <Layer>
          {shapes.map((s, i) =>
            s.type === 'rect'
              ? <Rect key={i} x={s.x} y={s.y} width={s.w} height={s.h} stroke={s.color} strokeWidth={3} />
              : <Circle key={i} x={s.x} y={s.y} radius={Math.abs(s.w)} stroke={s.color} strokeWidth={3} />
          )}
          {lines.map((ln, i) => (
            <Line key={i} points={ln.points} stroke={ln.color}
              strokeWidth={ln.tool === 'eraser' ? 20 : 3}
              tension={0.4} lineCap="round" lineJoin="round"
              globalCompositeOperation={ln.tool === 'eraser' ? 'destination-out' : 'source-over'} />
          ))}
        </Layer>
      </Stage>
    </div>
  );
};

export default Whiteboard;