import { useState, useEffect } from 'react';
import type { Stats, EntryPlayerStats, TeamEntryStats, OpeningDuel, UtilityPlayerStats, TeamUtilityStats, TeamBombStats, RoundBombEvent, TeamSiteStats, PlayerClutchStats, TeamClutchSituation, ClutchRound, GrenadeType } from "../helpers/types"

const GRENADE_LABELS: Record<GrenadeType, string> = {
    smoke: "Smoke",
    flashbang: "Flash",
    hegrenade: "HE",
    molotov: "Molotov",
    incendiary: "Incendiary",
    decoy: "Decoy",
};

function formatGrenadesThrown(grenadesThrown: Partial<Record<GrenadeType, number>>) {
    const entries = Object.entries(grenadesThrown) as [GrenadeType, number][];
    if (entries.length === 0) return "—";
    return entries.map(([type, count]) => `${GRENADE_LABELS[type]}: ${count}`).join(", ");
}

function formatElapsed(seconds?: number) {
    if (seconds === undefined) return "—";
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, "0")}`;
}

type AdvSubTab = "entries" | "utility" | "bomb" | "clutches";

function AdvancedSubNav({ active, setActive }: { active: AdvSubTab; setActive: (t: AdvSubTab) => void }) {
    const tabs: { key: AdvSubTab; label: string }[] = [
        { key: "entries", label: "Entries" },
        { key: "utility", label: "Utility" },
        { key: "bomb", label: "Plants & Retakes" },
        { key: "clutches", label: "Clutches" },
    ];
    return (
        <div className="pillNav">
            {tabs.map(t => (
                <div
                    key={t.key}
                    className={`pillItem ${active === t.key ? "pillActive" : ""}`}
                    onClick={() => setActive(t.key)}
                >
                    {t.label}
                </div>
            ))}
        </div>
    );
}

function TeamEntryCard({ team }: { team: TeamEntryStats }) {
    return (
        <div className="advCard">
            <h4>{team.team}</h4>
            <div className="metricGrid">
                <div><div className="metricLabel">T Entries (K/D)</div><div className="metricValue">{team.entryKillsT}/{team.entryDeathsT}</div></div>
                <div><div className="metricLabel">CT Entries (K/D)</div><div className="metricValue">{team.entryKillsCT}/{team.entryDeathsCT}</div></div>
                <div><div className="metricLabel">Traded entry deaths</div><div className="metricValue">{team.tradedEntryDeaths}</div></div>
                <div><div className="metricLabel">Trade opportunities</div><div className="metricValue">{team.tradeOpportunities}</div></div>
                <div><div className="metricLabel">Trades converted</div><div className="metricValue">{team.tradesConverted}</div></div>
                <div><div className="metricLabel">Missed opportunities</div><div className="metricValue">{team.missedTradeOpportunities}</div></div>
                <div><div className="metricLabel">Untradable deaths</div><div className="metricValue">{team.untradableDeaths}</div></div>
            </div>
        </div>
    );
}

function EntryTable({ teamName, rows }: { teamName: string; rows: EntryPlayerStats[] }) {
    return (
        <div className="advCard">
            <h4>{teamName}</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Player</td>
                        <td>T Entries (K/D)</td>
                        <td>CT Entries (K/D)</td>
                        <td>Traded</td>
                        <td>Trade Opp</td>
                        <td>Converted</td>
                        <td>Missed</td>
                        <td>Untradable</td>
                    </tr>
                </thead>
                <tbody>
                    {rows.map(p => (
                        <tr key={p.playerId}>
                            <td className="playerName">{p.name}</td>
                            <td className="numCell">{p.entryKillsT}/{p.entryDeathsT}</td>
                            <td className="numCell">{p.entryKillsCT}/{p.entryDeathsCT}</td>
                            <td className="numCell">{p.tradedEntryDeaths}</td>
                            <td className="numCell">{p.tradeOpportunities}</td>
                            <td className="numCell">{p.tradesConverted}</td>
                            <td className="numCell">{p.missedTradeOpportunities}</td>
                            <td className="numCell">{p.untradableDeaths}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function OpeningDuelTable({ rounds }: { rounds: OpeningDuel[] }) {
    return (
        <div className="advCard">
            <h4>Opening picks</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Round</td>
                        <td>T Buy</td>
                        <td>CT Buy</td>
                        <td title="Time elapsed in the round since freeze time ended">Time Elapsed</td>
                        <td>Killer</td>
                        <td>Side</td>
                        <td>Victim</td>
                        <td>Traded</td>
                        <td>Round Winner</td>
                    </tr>
                </thead>
                <tbody>
                    {rounds.map(r => (
                        <tr key={r.roundNo}>
                            <td>{r.roundNo}</td>
                            <td>{r.tBuyType ?? "—"}</td>
                            <td>{r.ctBuyType ?? "—"}</td>
                            <td className="numCell">{formatElapsed(r.elapsedSeconds)}</td>
                            <td className="playerName">{r.killerName ?? "—"} <span style={{ opacity: 0.6 }}>({r.killerTeam ?? "—"})</span></td>
                            <td>{r.killerSide ?? "—"}</td>
                            <td className="playerName">{r.victimName ?? "—"} <span style={{ opacity: 0.6 }}>({r.victimTeam ?? "—"})</span></td>
                            <td>{r.traded ? "Yes" : "No"}</td>
                            <td>{r.winningTeam ?? "—"}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function EntriesPanel({ matchId, teamAName, teamBName }: { matchId: number; teamAName: string; teamBName: string }) {
    const [players, setPlayers] = useState<EntryPlayerStats[]>([]);
    const [teams, setTeams] = useState<TeamEntryStats[]>([]);
    const [rounds, setRounds] = useState<OpeningDuel[]>([]);

    useEffect(() => {
        fetch(`http://localhost:4000/api/match/entries/${matchId}`, { credentials: 'include' })
            .then(r => r.json())
            .then((data: { players: EntryPlayerStats[]; teams: TeamEntryStats[]; rounds: OpeningDuel[] }) => {
                setPlayers(data.players ?? []);
                setTeams(data.teams ?? []);
                setRounds(data.rounds ?? []);
            })
            .catch(() => {});
    }, [matchId]);

    return (
        <>
            <div className="advHeading">Team totals</div>
            <div className="advGrid2col">
                {teams.map(t => <TeamEntryCard team={t} key={t.team} />)}
            </div>

            <div className="advHeading">Entry duels</div>
            <div className="advGrid2col">
                <EntryTable teamName={teamAName} rows={players.filter(p => p.team === teamAName)} />
                <EntryTable teamName={teamBName} rows={players.filter(p => p.team === teamBName)} />
            </div>

            <div className="advHeading">Round-by-round</div>
            <OpeningDuelTable rounds={rounds} />
        </>
    );
}

function TeamUtilityCard({ team }: { team: TeamUtilityStats }) {
    return (
        <div className="advCard">
            <h4>{team.team}</h4>
            <div className="metricGrid">
                <div><div className="metricLabel">Utility Dmg</div><div className="metricValue">{team.utilityDamage}</div></div>
                <div><div className="metricLabel">HE Dmg</div><div className="metricValue">{team.heDamage}</div></div>
                <div><div className="metricLabel">Fire Dmg</div><div className="metricValue">{team.fireDamage}</div></div>
                <div><div className="metricLabel">Flashes Thrown</div><div className="metricValue">{team.flashesThrown}</div></div>
                <div><div className="metricLabel">Enemies Flashed</div><div className="metricValue">{team.enemiesFlashed}</div></div>
                <div><div className="metricLabel">Avg Blind</div><div className="metricValue">{team.avgBlindTime.toFixed(1)}s</div></div>
                <div><div className="metricLabel">Flash Assists</div><div className="metricValue">{team.flashAssists}</div></div>
                <div><div className="metricLabel">Flashes → Kill</div><div className="metricValue">{team.flashesToKill}</div></div>
            </div>
            <div style={{ marginTop: 8, fontSize: 12, opacity: 0.75 }}>{formatGrenadesThrown(team.grenadesThrown)}</div>
        </div>
    );
}

function UtilityTable({ teamName, rows }: { teamName: string; rows: UtilityPlayerStats[] }) {
    return (
        <div className="advCard">
            <h4>{teamName}</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Player</td>
                        <td>Util Dmg</td>
                        <td>HE Dmg</td>
                        <td>Fire Dmg</td>
                        <td>Flashes</td>
                        <td>Enemies Flashed</td>
                        <td>Avg Blind</td>
                        <td>Flash Assists</td>
                    </tr>
                </thead>
                <tbody>
                    {rows.map(p => (
                        <tr key={p.playerId}>
                            <td className="playerName">{p.name}</td>
                            <td className="numCell">{p.utilityDamage}</td>
                            <td className="numCell">{p.heDamage}</td>
                            <td className="numCell">{p.fireDamage}</td>
                            <td className="numCell">{p.flashesThrown}</td>
                            <td className="numCell">{p.enemiesFlashed}</td>
                            <td className="numCell">{p.avgBlindTime.toFixed(1)}s</td>
                            <td className="numCell">{p.flashAssists}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function UtilityPanel({ matchId, teamAName, teamBName }: { matchId: number; teamAName: string; teamBName: string }) {
    const [players, setPlayers] = useState<UtilityPlayerStats[]>([]);
    const [teams, setTeams] = useState<TeamUtilityStats[]>([]);

    useEffect(() => {
        fetch(`http://localhost:4000/api/match/utility/${matchId}`, { credentials: 'include' })
            .then(r => r.json())
            .then((data: { players: UtilityPlayerStats[]; teams: TeamUtilityStats[] }) => {
                setPlayers(data.players ?? []);
                setTeams(data.teams ?? []);
            })
            .catch(() => {});
    }, [matchId]);

    return (
        <>
            <div className="advHeading">Team totals</div>
            <div className="advGrid2col">
                {teams.map(t => <TeamUtilityCard team={t} key={t.team} />)}
            </div>

            <div className="advHeading">Utility usage</div>
            <div className="advGrid2col">
                <UtilityTable teamName={teamAName} rows={players.filter(p => p.team === teamAName)} />
                <UtilityTable teamName={teamBName} rows={players.filter(p => p.team === teamBName)} />
            </div>
        </>
    );
}

const OUTCOME_LABELS: Record<string, string> = {
    retake_success: "Retake success",
    save: "CT save",
    retake_failed: "Retake failed",
    t_save: "T save",
    t_eliminated: "T eliminated",
    t_win_no_plant: "T win (no plant)",
};

function TeamBombCard({ team }: { team: TeamBombStats }) {
    return (
        <div className="advCard">
            <h4>{team.team}</h4>
            <div className="metricGrid">
                <div><div className="metricLabel">Plants</div><div className="metricValue">{team.plants}</div></div>
                <div><div className="metricLabel">Plant win%</div><div className="metricValue">{team.plantWinPct.toFixed(0)}%</div></div>
                <div><div className="metricLabel">Defuses</div><div className="metricValue">{team.defuses}</div></div>
                <div><div className="metricLabel">Retake attempts</div><div className="metricValue">{team.retakeAttempts}</div></div>
                <div><div className="metricLabel">Retake success%</div><div className="metricValue">{team.retakeSuccessPct.toFixed(0)}%</div></div>
                <div><div className="metricLabel">Saves</div><div className="metricValue">{team.saves}</div></div>
                <div><div className="metricLabel">Holds</div><div className="metricValue">{team.holds}</div></div>
                <div><div className="metricLabel">Hold%</div><div className="metricValue">{team.holdPct.toFixed(0)}%</div></div>
            </div>
        </div>
    );
}

function SiteBarRow({ label, pct, valueLabel, color, tooltip, refLinePct }: {
    label: string; pct: number; valueLabel: string; color: string; tooltip: string; refLinePct?: number;
}) {
    return (
        <div className="siteBarRow">
            <span className="siteBarSiteLabel">{label}</span>
            <div className="siteBarTrack" title={tooltip}>
                {refLinePct !== undefined && <div className="siteBarRefLine" style={{ left: `${refLinePct}%` }} />}
                <div className="siteBarFill" style={{ width: `${pct}%`, background: color }} />
            </div>
            <span className="siteBarValue">{valueLabel}</span>
        </div>
    );
}

// A 100%-stacked bar: won/lost proportion of this site's plants. Bar always
// spans the full track (won% + lost% == 100 whenever plants > 0), so the
// track's own rounded corners double as the segment end-caps — no clipping
// edge case to handle.
function SiteStackedBarRow({ site }: { site: TeamSiteStats }) {
    const wonPct = site.plants > 0 ? (site.plantWins / site.plants) * 100 : 0;
    const lostPct = site.plants > 0 ? (site.plantLosses / site.plants) * 100 : 0;
    return (
        <div className="siteBarRow">
            <span className="siteBarSiteLabel">{site.site}</span>
            <div
                className="siteStackTrack"
                title={`Site ${site.site}: ${site.plants} plant${site.plants === 1 ? "" : "s"} — ${site.plantWins} won, ${site.plantLosses} lost to retake`}
            >
                {wonPct > 0 && <div className="siteStackSeg" style={{ flexBasis: `${wonPct}%`, background: "var(--good)" }} />}
                {wonPct > 0 && lostPct > 0 && <div className="siteStackGap" />}
                {lostPct > 0 && <div className="siteStackSeg" style={{ flexBasis: `${lostPct}%`, background: "var(--bad)" }} />}
            </div>
            <span className="siteBarValue">
                {site.plants === 0 ? "No plants" : `${site.plants} (${site.plantWins}W/${site.plantLosses}L)`}
            </span>
        </div>
    );
}

function TeamSiteCard({ team, sites }: { team: string; sites: TeamSiteStats[] }) {
    return (
        <div className="advCard">
            <h4>{team}</h4>
            <div className="advSubHeading">Plants by site — won / lost to retake</div>
            {sites.map(s => <SiteStackedBarRow key={s.site} site={s} />)}

            <div className="advSubHeading">Retake success % by site (as CT)</div>
            {sites.map(s => (
                <SiteBarRow
                    key={s.site}
                    label={s.site}
                    pct={s.retakeAttempts > 0 ? s.retakeSuccessPct : 0}
                    valueLabel={s.retakeAttempts > 0 ? `${s.retakeSuccessPct.toFixed(0)}% (${s.retakeSuccesses}/${s.retakeAttempts})` : "No attempts"}
                    color="var(--ct-side)"
                    tooltip={
                        s.retakeAttempts > 0
                            ? `Site ${s.site}: ${s.retakeSuccesses} of ${s.retakeAttempts} retakes won (${s.retakeSuccessPct.toFixed(0)}%)`
                            : `Site ${s.site}: bomb was never planted here against ${team}`
                    }
                    refLinePct={50}
                />
            ))}
        </div>
    );
}

function BombRoundTable({ rounds }: { rounds: RoundBombEvent[] }) {
    return (
        <div className="advCard">
            <h4>Round-by-round</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Round</td>
                        <td>Site</td>
                        <td>Planter</td>
                        <td>Defuser</td>
                        <td>Winner</td>
                        <td>Outcome</td>
                    </tr>
                </thead>
                <tbody>
                    {rounds.map(r => (
                        <tr key={r.roundNo}>
                            <td>{r.roundNo}</td>
                            <td>{r.site ?? "—"}</td>
                            <td className="playerName">{r.planter ?? "—"}</td>
                            <td className="playerName">{r.defuser ?? "—"}</td>
                            <td>{r.winningTeam ?? "—"}</td>
                            <td>{OUTCOME_LABELS[r.outcome] ?? "—"}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function BombPanel({ matchId, teamAName, teamBName }: { matchId: number; teamAName: string; teamBName: string }) {
    const [teams, setTeams] = useState<TeamBombStats[]>([]);
    const [rounds, setRounds] = useState<RoundBombEvent[]>([]);
    const [teamSites, setTeamSites] = useState<TeamSiteStats[]>([]);

    useEffect(() => {
        fetch(`http://localhost:4000/api/match/plantsRetakes/${matchId}`, { credentials: 'include' })
            .then(r => r.json())
            .then((data: { teams: TeamBombStats[]; rounds: RoundBombEvent[]; teamSites: TeamSiteStats[] }) => {
                setTeams(data.teams ?? []);
                setRounds(data.rounds ?? []);
                setTeamSites(data.teamSites ?? []);
            })
            .catch(() => {});
    }, [matchId]);

    return (
        <>
            <div className="advHeading">Plants & retakes</div>
            <div className="advGrid2col">
                {teams.map(t => <TeamBombCard team={t} key={t.team} />)}
            </div>

            <div className="advHeading">By site</div>
            <div className="advGrid2col">
                <TeamSiteCard team={teamAName} sites={teamSites.filter(s => s.team === teamAName)} />
                <TeamSiteCard team={teamBName} sites={teamSites.filter(s => s.team === teamBName)} />
            </div>

            <BombRoundTable rounds={rounds} />
        </>
    );
}

function ClutchPlayerTable({ teamName, rows }: { teamName: string; rows: PlayerClutchStats[] }) {
    return (
        <div className="advCard">
            <h4>{teamName}</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Player</td>
                        <td>Attempts</td>
                        <td>Wins</td>
                        <td>Win%</td>
                    </tr>
                </thead>
                <tbody>
                    {rows.map(p => (
                        <tr key={p.playerId}>
                            <td className="playerName">{p.name}</td>
                            <td className="numCell">{p.attempts}</td>
                            <td className="numCell">{p.wins}</td>
                            <td className="numCell">{p.winPct.toFixed(0)}%</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function TeamClutchCard({ team, situations }: { team: string; situations: TeamClutchSituation[] }) {
    const totalAttempts = situations.reduce((sum, s) => sum + s.attempts, 0);
    const totalWins = situations.reduce((sum, s) => sum + s.wins, 0);
    return (
        <div className="advCard">
            <h4>{team}</h4>
            <div className="metricGrid">
                <div><div className="metricLabel">Attempts</div><div className="metricValue">{totalAttempts}</div></div>
                <div><div className="metricLabel">Wins</div><div className="metricValue">{totalWins}</div></div>
                <div><div className="metricLabel">Win%</div><div className="metricValue">{totalAttempts > 0 ? ((totalWins / totalAttempts) * 100).toFixed(0) : 0}%</div></div>
            </div>
            <div className="advSubHeading">By situation</div>
            {situations.map(s => (
                <SiteBarRow
                    key={s.opponents}
                    label={`1v${s.opponents}`}
                    pct={s.attempts > 0 ? s.winPct : 0}
                    valueLabel={s.attempts > 0 ? `${s.wins}/${s.attempts} (${s.winPct.toFixed(0)}%)` : "No attempts"}
                    color="var(--good)"
                    tooltip={
                        s.attempts > 0
                            ? `1v${s.opponents}: won ${s.wins} of ${s.attempts} (${s.winPct.toFixed(0)}%)`
                            : `${team} never faced a 1v${s.opponents}`
                    }
                    refLinePct={50}
                />
            ))}
        </div>
    );
}

function ClutchRoundTable({ rounds }: { rounds: ClutchRound[] }) {
    return (
        <div className="advCard">
            <h4>Round-by-round</h4>
            <table className="advTable">
                <thead>
                    <tr>
                        <td>Round</td>
                        <td>Player</td>
                        <td>Team</td>
                        <td>Side</td>
                        <td>Situation</td>
                        <td>Result</td>
                    </tr>
                </thead>
                <tbody>
                    {rounds.map(r => (
                        <tr key={r.roundNo}>
                            <td>{r.roundNo}</td>
                            <td className="playerName">{r.playerName}</td>
                            <td>{r.team}</td>
                            <td>{r.side ?? "—"}</td>
                            <td>1v{r.opponents}</td>
                            <td className={r.won ? "pos" : "neg"}>{r.won ? "Won" : "Lost"}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function ClutchesPanel({ matchId, teamAName, teamBName }: { matchId: number; teamAName: string; teamBName: string }) {
    const [players, setPlayers] = useState<PlayerClutchStats[]>([]);
    const [situations, setSituations] = useState<TeamClutchSituation[]>([]);
    const [rounds, setRounds] = useState<ClutchRound[]>([]);

    useEffect(() => {
        fetch(`http://localhost:4000/api/match/clutches/${matchId}`, { credentials: 'include' })
            .then(r => r.json())
            .then((data: { players: PlayerClutchStats[]; situations: TeamClutchSituation[]; rounds: ClutchRound[] }) => {
                setPlayers(data.players ?? []);
                setSituations(data.situations ?? []);
                setRounds(data.rounds ?? []);
            })
            .catch(() => {});
    }, [matchId]);

    return (
        <>
            <div className="advHeading">Clutches by situation</div>
            <div className="advGrid2col">
                <TeamClutchCard team={teamAName} situations={situations.filter(s => s.team === teamAName)} />
                <TeamClutchCard team={teamBName} situations={situations.filter(s => s.team === teamBName)} />
            </div>

            <div className="advHeading">Player breakdown</div>
            <div className="advGrid2col">
                <ClutchPlayerTable teamName={teamAName} rows={players.filter(p => p.team === teamAName)} />
                <ClutchPlayerTable teamName={teamBName} rows={players.filter(p => p.team === teamBName)} />
            </div>

            <div className="advHeading">Round-by-round</div>
            <ClutchRoundTable rounds={rounds} />
        </>
    );
}

function AdvancedStatsPanel({ TeamA, TeamB, matchId }: { TeamA: Stats; TeamB: Stats; matchId: number }) {
    const [sub, setSub] = useState<AdvSubTab>("entries");
    return (
        <div className="advSection">
            <AdvancedSubNav active={sub} setActive={setSub} />
            {sub === "entries" && <EntriesPanel matchId={matchId} teamAName={TeamA.Clanname} teamBName={TeamB.Clanname} />}
            {sub === "utility" && <UtilityPanel matchId={matchId} teamAName={TeamA.Clanname} teamBName={TeamB.Clanname} />}
            {sub === "bomb" && <BombPanel matchId={matchId} teamAName={TeamA.Clanname} teamBName={TeamB.Clanname} />}
            {sub === "clutches" && <ClutchesPanel matchId={matchId} teamAName={TeamA.Clanname} teamBName={TeamB.Clanname} />}
        </div>
    );
}
export default AdvancedStatsPanel;
