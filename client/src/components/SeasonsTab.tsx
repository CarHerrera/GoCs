import React, { useState, useEffect } from 'react';
interface Season {
  id: number;
  name: string;
  startDate: string;
  endDate: string;
  notes: string;
}

const SeasonsSection: React.FC = () => {
  const [seasons, setSeasons] = useState<Season[]>([]);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState({ name: '', startDate: '', endDate: '', notes: '' });
  const [creating, setCreating] = useState(false);

  const fetchSeasons = () =>
    fetch('http://localhost:4000/api/team/seasons', { credentials: 'include' })
      .then(r => r.json()).then(setSeasons).catch(() => {});

  useEffect(() => { fetchSeasons(); }, []);

  const saveNew = async () => {
    await fetch('http://localhost:4000/api/team/seasons', {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    });
    setForm({ name: '', startDate: '', endDate: '', notes: '' });
    setCreating(false);
    fetchSeasons();
  };

  const saveEdit = async (id: number) => {
    const season = seasons.find(s => s.id === id)!;
    await fetch(`http://localhost:4000/api/team/seasons/${id}`, {
      method: 'PUT', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(season),
    });
    setEditingId(null);
  };

  const updateField = (id: number, field: keyof Season, value: string) =>
    setSeasons(ss => ss.map(s => s.id === id ? { ...s, [field]: value } : s));

  return (
    <div className="settings-section">
      <div className="settings-files-header">
        <p className="settings-section-desc">Create and manage seasons.</p>
        <button className="settings-upload-btn" onClick={() => setCreating(true)}>+ New Season</button>
      </div>

      {creating && (
        <div className="settings-season-form">
          <input className="settings-input" placeholder="Season name"
            value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          <input className="settings-input" type="date"
            value={form.startDate} onChange={e => setForm(f => ({ ...f, startDate: e.target.value }))} />
          <input className="settings-input" type="date"
            value={form.endDate} onChange={e => setForm(f => ({ ...f, endDate: e.target.value }))} />
          <input className="settings-input" placeholder="Notes (optional)"
            value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} />
          <div className="settings-form-actions">
            <button className="settings-save-btn" onClick={saveNew}>Save</button>
            <button className="settings-cancel-btn" onClick={() => setCreating(false)}>Cancel</button>
          </div>
        </div>
      )}

      <div className="settings-season-list">
        {seasons.map(s => (
          <div key={s.id} className="settings-season-item">
            {editingId === s.id ? (
              <>
                <input className="settings-input" value={s.name}
                  onChange={e => updateField(s.id, 'name', e.target.value)} />
                <input className="settings-input" type="date" value={s.startDate}
                  onChange={e => updateField(s.id, 'startDate', e.target.value)} />
                <input className="settings-input" type="date" value={s.endDate}
                  onChange={e => updateField(s.id, 'endDate', e.target.value)} />
                <div className="settings-form-actions">
                  <button className="settings-save-btn" onClick={() => saveEdit(s.id)}>Save</button>
                  <button className="settings-cancel-btn" onClick={() => setEditingId(null)}>Cancel</button>
                </div>
              </>
            ) : (
              <>
                <div className="settings-season-info">
                  <span className="settings-season-name">{s.name}</span>
                  <span className="settings-season-dates">{s.startDate} → {s.endDate || 'ongoing'}</span>
                </div>
                <button className="settings-edit-btn" onClick={() => setEditingId(s.id)}>Edit</button>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default SeasonsSection;