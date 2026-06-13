export const WeaponType = {
    // Pistols
    P2000: 1,
    Glock: 2,
    P250: 3,
    Deagle: 4,
    Fiveseven: 5,
    Elite: 6,
    Tec9: 7,
    CZ: 8,
    UspSilencer: 9,
    Revolver: 10,

    // SMGs
    Mp7: 101,
    Mp9: 102,
    Bizon: 103,
    Mac10: 104,
    Ump45: 105,
    P90: 106,
    Mp5sd: 107,

    // Heavy
    Sawedoff: 201,
    Nova: 202,
    Swag7: 203,
    Xm1014: 204,
    M249: 205,
    Negev: 206,

    // Rifles
    Galilar: 301,
    Famas: 302,
    Ak47: 303,
    M4a1: 304,
    M4a1Silencer: 305,
    Ssg08: 306,
    Sg556: 307,
    Aug: 308,
    Awp: 309,
    Scar20: 310,
    G3sg1: 311,

    // Gear & Utility
    C4: 404,
    Knife: 405,
    WorldFallback: 407,

    // Grenades
    Decoy: 501,
    Molotov: 502,
    Incgrenade: 503,
    Flashbang: 504,
    Smokegrenade: 505,
    Hegrenade: 506,

    // Default
    None: 0,
} as const;
export type WeaponTypeValue = typeof WeaponType[keyof typeof WeaponType];
const WeaponSvgMap: Record<number, string> = {
    [WeaponType.P2000]: 'hkp2000.svg',
    [WeaponType.Glock]: 'glock.svg',
    [WeaponType.P250]: 'p250.svg',
    [WeaponType.Deagle]: 'deagle.svg',
    [WeaponType.Fiveseven]: 'fiveseven.svg',
    [WeaponType.Elite]: 'elite.svg',
    [WeaponType.Tec9]: 'tec9.svg',
    [WeaponType.CZ]: 'cz75a.svg',
    [WeaponType.UspSilencer]: 'usp_silencer.svg',
    [WeaponType.Revolver]: 'revolver.svg',

    [WeaponType.Mp7]: 'mp7.svg',
    [WeaponType.Mp9]: 'mp9.svg',
    [WeaponType.Bizon]: 'bizon.svg',
    [WeaponType.Mac10]: 'mac10.svg',
    [WeaponType.Ump45]: 'ump45.svg',
    [WeaponType.P90]: 'p90.svg',
    [WeaponType.Mp5sd]: 'mp5sd.svg',

    [WeaponType.Sawedoff]: 'sawedoff.svg',
    [WeaponType.Nova]: 'nova.svg',
    [WeaponType.Swag7]: 'mag7.svg',
    [WeaponType.Xm1014]: 'xm1014.svg',
    [WeaponType.M249]: 'm249.svg',
    [WeaponType.Negev]: 'negev.svg',

    [WeaponType.Galilar]: 'galilar.svg',
    [WeaponType.Famas]: 'famas.svg',
    [WeaponType.Ak47]: 'ak47.svg',
    [WeaponType.M4a1]: 'm4a1.svg',
    [WeaponType.M4a1Silencer]: 'm4a1_silencer.svg',
    [WeaponType.Ssg08]: 'ssg08.svg',
    [WeaponType.Sg556]: 'sg556.svg',
    [WeaponType.Aug]: 'aug.svg',
    [WeaponType.Awp]: 'awp.svg',
    [WeaponType.Scar20]: 'scar20.svg',
    [WeaponType.G3sg1]: 'g3sg1.svg',

    [WeaponType.C4]: 'c4.svg',
    [WeaponType.Knife]: 'knife.svg',
    [WeaponType.WorldFallback]: 'world.svg',

    [WeaponType.Decoy]: 'decoy.svg',
    [WeaponType.Molotov]: 'molotov.svg',
    [WeaponType.Incgrenade]: 'incgrenade.svg',
    [WeaponType.Flashbang]: 'flashbang.svg',
    [WeaponType.Smokegrenade]: 'smokegrenade.svg',
    [WeaponType.Hegrenade]: 'hegrenade.svg',
    
    [WeaponType.None]: 'world.svg',
};
export function type_to_svg(gunType: WeaponTypeValue | number): string {
    // Look up file name in the map, default to 'world.svg' if not found
    const file = WeaponSvgMap[gunType] || 'world.svg';
    return `equipment/${file}`;
}
