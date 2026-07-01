import React, { type ReactElement } from 'react';

export interface StatColumn {
  label: string;
  value: string | number;
  divider: string;
  valueColor: string;
}

export const createStatCol = (
  label: string,
  value: string | number,
  idx: number,
  highlight: boolean
): StatColumn => ({
  label,
  value,
  divider: idx === 0 ? 'none' : '1px solid rgba(255,255,255,0.06)',
  valueColor: highlight ? '#3b82f6' : '#ffffff',
});

interface StatGridProps {
  stats: StatColumn[];
}

const StatGrid: React.FC<StatGridProps> = ({ stats }): ReactElement => (
  <div className="stat-grid">
    {stats.map((stat, idx): ReactElement => (
      <div
        key={idx}
        className="stat-column"
        style={{ borderLeft: stat.divider }}
      >
        <div className={`stat-column-value ${stat.valueColor === '#3b82f6' ? 'highlight' : ''}`}>
          {stat.value}
        </div>
        <div className="stat-column-label">{stat.label}</div>
      </div>
    ))}
  </div>
);

export default StatGrid;