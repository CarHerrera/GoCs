export function type_to_svg(gunType:number):string{
    let path = 'equipment/'
    switch(gunType){
        case 1:
            path += 'hkp2000.svg'
            break
        case 2:
            path += 'glock.svg'
            break
        case 3:
            path += 'p250.svg'
            break
        case 4:
            path += 'deagle.svg'
            break
        case 5:
            path += 'fiveseven.svg'
            break
        case 6:
            path += 'elite.svg'
            break
        case 7:
            path += 'tec9.svg'
            break
        case 8: 
            path += 'cz75a.svg'
            break
        case 9: 
            path += 'usp_silencer.svg'
            break
        case 10:
            path += 'revolver.svg'
            break
        case 101: 
            path += 'mp7.svg'
            break
        case 102: 
            path += 'mp9.svg'
            break
        case 103:
            path += 'bizon.svg'
            break
        case 104: 
            path += 'mac10.svg'
            break
        case 105:
            path += 'ump45.svg'
            break
        case 106: 
            path += 'p90.svg'
            break
        case 107:
            path +='mp5sd.svg'
            break
        case 201:
            path +='sawedoff.svg'
            break
        case 202:
            path +='nova.svg'
            break
        case 203:
            path +='mag7.svg'
            break
        case 204: 
            path +='xm1014.svg'
            break
        case 205:
            path +='m249.svg'
            break
        case 206:
            path += 'negev.svg'
            break
        case 301:
            path += 'galilar.svg'
            break
        case 302: 
            path += 'famas.svg'
            break
        case 303:
            path += 'ak47.svg'
            break
        case 304:
            path += 'm4a1.svg'
            break
        case 305:
            path += 'm4a1_silencer.svg'
            break
        case 306:
            path +='ssg08.svg'
            break
        case 307:
            path += 'sg556.svg'
            break
        case 308: 
            path += 'aug.svg'
            break
        case 309:
            path += 'awp.svg'
            break
        case 310:
            path +='scar20.svg'
            break
        case 311:
            path +='g3sg1.svg'
            break
        case 404:
            path += 'c4.svg'
            break
        case 405:
            path += "knife.svg"
            break
        case 501:
            path += 'decoy.svg'
            break
        case 502: 
            path +='molotov.svg'
            break
        case 503:
            path +='incgrenade.svg'
            break
        case 504:
            path +='flashbang.svg'
            break
        case 505:
            path +='smokegrenade.svg'
            break
        case 506:
            path += 'hegrenade.svg'
            break
        case 0:
        case 407:
        default:
            path += 'world.svg'
        break
    }
    return path;
}

