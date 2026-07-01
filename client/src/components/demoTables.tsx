import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

interface fileInfo{
    filename: string,
    date: string,
    savedate: string,
    filesize: number,
    map: string
    parsed: boolean
    stats: boolean
}
function DemoTable() {
    const [files, setFiles] = useState<fileInfo[]>([]);
    async function getDemos(){
        return fetch('http://localhost:4000/AllDemos', {
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
            console.log(data);
            return data;
        })
    }
    useEffect(() => {
        let ignore = false;
        async function getFetch(){
            const data = await getDemos();
            if(!ignore){
                setFiles(data)
            }
        }
        getFetch()
        return () => { ignore = true}
    }, [])
    const tableRows = files.map((x,i) => {
        let file = x.filename
        let map = x.map 
        let date = x.date
        let parsed = x.parsed
        let stats = x.stats
        let saved = x.savedate
        if (file[0] == "\""){
            file = x.filename.substring(1, file.length-1)
        }
        if (map[0] == "\""){
            map = x.map.substring(1, map.length-1)
        }
        if (date[0] == "\""){
            date = x.date.substring(1, date.length-1)
        }
        if (saved[0] == "\""){
            saved = x.savedate.substring(1, saved.length-1)
        }
        console.log(x)
        console.log(stats)
        return <tr key={i}><td>{file}</td><td>{saved}</td><td>{date}</td><td>{map}</td><td>{stats ? "true" : "false" }</td><td>{parsed ? "true" : "false"}</td>
        <td><Link to={`/advancedStats?file=${file}&map=${map}`}>Stats</Link></td></tr>
    })
    return <>
        <div >
            <table style={{width:"100%"}}>
                <thead>
                    <tr>
                        <th>File Name</th>
                        <th>Day Played</th>
                        <th>Date Uploaded</th>
                        <th>Map</th>
                        <th>HasStats?</th>
                        <th>Parsed?</th>
                        <th>Advanved Stats</th>
                    </tr>
                </thead>
                <tbody>
                    {tableRows}
                </tbody>
            </table>
        </div>
        
    </>
}

export default DemoTable