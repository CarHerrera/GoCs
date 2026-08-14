export interface Stats{
    ID: number,
    Clanname: string,
    Endscore: number,
    TScore: number,
    CTScore: number,
    Playing: Record<string, Player>
}
export interface Player{
    name: string,
    ID: string,
    stats: PlayerStats
}
export interface PlayerStats {
    kills: number;
    assists: number;
    deaths: number;
    hs: number;
    "1k": number;
    "2k": number;
    "3k": number;
    "4k": number;
    "5k": number;
    adr: number;
    dmg: number;
    ud: number;
    rating: number;
    kast: number;
}

    // entry_kills: number;
    // entry_deaths: number;
// traded_deaths: number;
//     trade_kills: number;
//     clutch_win: number;
//     clutch_count: number;
//     flash_assists: number;
//     team_flashes: number;
//     enemies_flashed: number;
//     he_damage: number;
//     fire_damage: number;

export interface EntryPlayerStats {
    playerId: string;
    name: string;
    team: string;
    entryKillsT: number;
    entryDeathsT: number;
    entryKillsCT: number;
    entryDeathsCT: number;
    tradedEntryDeaths: number;
    // Trade fields cover every death in the match, not just entry deaths.
    tradeOpportunities: number;
    tradesConverted: number;
    missedTradeOpportunities: number;
    untradableDeaths: number;
}

export interface TeamEntryStats {
    team: string;
    entryKillsT: number;
    entryDeathsT: number;
    entryKillsCT: number;
    entryDeathsCT: number;
    tradedEntryDeaths: number;
    tradeOpportunities: number;
    tradesConverted: number;
    missedTradeOpportunities: number;
    untradableDeaths: number;
}

export type BuyType = "Pistol" | "Eco" | "Force" | "Full";

export interface OpeningDuel {
    roundNo: number;
    elapsedSeconds?: number;
    killerName?: string;
    killerTeam?: string;
    killerSide?: "T" | "CT";
    victimName?: string;
    victimTeam?: string;
    victimSide?: "T" | "CT";
    traded: boolean;
    tBuyType?: BuyType;
    ctBuyType?: BuyType;
    winningTeam?: string;
}

export interface EntryStatsResponse {
    players: EntryPlayerStats[];
    teams: TeamEntryStats[];
    rounds: OpeningDuel[];
}

export type GrenadeType = "decoy" | "molotov" | "incendiary" | "flashbang" | "smoke" | "hegrenade";

export interface UtilityPlayerStats {
    playerId: string;
    name: string;
    team: string;
    utilityDamage: number;
    heDamage: number;
    fireDamage: number;
    flashAssists: number;
    teamFlashed: number;
    enemiesFlashed: number;
    flashesThrown: number;
    avgBlindTime: number;
    flashesToKill: number;
    grenadesThrown: Partial<Record<GrenadeType, number>>;
}

export interface TeamUtilityStats {
    team: string;
    utilityDamage: number;
    heDamage: number;
    fireDamage: number;
    flashAssists: number;
    teamFlashed: number;
    enemiesFlashed: number;
    flashesThrown: number;
    avgBlindTime: number;
    flashesToKill: number;
    grenadesThrown: Partial<Record<GrenadeType, number>>;
}

export interface UtilityStatsResponse {
    players: UtilityPlayerStats[];
    teams: TeamUtilityStats[];
}

export interface TeamBombStats {
    team: string;
    plants: number;
    plantRoundWins: number;
    plantWinPct: number;
    defuses: number;
    bombExplodes: number;
    retakeAttempts: number;
    retakeSuccesses: number;
    retakeSuccessPct: number;
    saves: number;
    // Holds: rounds played as CT where the bomb never got planted and the
    // team won. ctRoundsPlayed is the denominator for holdPct.
    holds: number;
    holdPct: number;
    ctRoundsPlayed: number;
}

// TeamSiteStats is one team's plant/retake record at one bombsite.
// plantWins + plantLosses always equals plants (won after planting there vs
// got retaken). retakeAttempts/retakeSuccesses are this team's record
// defending that site as CT.
export interface TeamSiteStats {
    team: string;
    site: "A" | "B";
    plants: number;
    plantWins: number;
    plantLosses: number;
    plantWinPct: number;
    retakeAttempts: number;
    retakeSuccesses: number;
    retakeSuccessPct: number;
}

export type BombRoundOutcome =
    | "retake_success"
    | "save"
    | "retake_failed"
    | "t_save"
    | "t_eliminated"
    | "t_win_no_plant";

export interface RoundBombEvent {
    roundNo: number;
    site?: "A" | "B";
    plantingTeam?: string;
    planter?: string;
    defendingTeam: string;
    defuser?: string;
    bombPlanted: boolean;
    bombDefused: boolean;
    bombExploded: boolean;
    winningTeam?: string;
    outcome: BombRoundOutcome | "";
}

export interface PlantsRetakesResponse {
    teams: TeamBombStats[];
    rounds: RoundBombEvent[];
    teamSites: TeamSiteStats[];
}

export interface PlayerClutchStats {
    playerId: string;
    name: string;
    team: string;
    attempts: number;
    wins: number;
    winPct: number;
}

// TeamClutchSituation is one team's record in a given "1vN" situation.
// Always present for opponents 1-5 per team, zero-filled if it never came up.
export interface TeamClutchSituation {
    team: string;
    opponents: 1 | 2 | 3 | 4 | 5;
    attempts: number;
    wins: number;
    winPct: number;
}

export interface ClutchRound {
    roundNo: number;
    playerName: string;
    team: string;
    side?: "T" | "CT";
    opponents: number;
    won: boolean;
}

export interface ClutchStatsResponse {
    players: PlayerClutchStats[];
    situations: TeamClutchSituation[];
    rounds: ClutchRound[];
}
