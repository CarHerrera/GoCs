import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

interface fileInfo{
    filename: string,
    date: string,
    filesize: number,
    map: string
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
            console.log(response)
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
        if (file[0] == "\""){
            file = x.filename.substring(1, file.length-1)
        }
        if (map[0] == "\""){
            map = x.map.substring(1, map.length-1)
        }
        if (date[0] == "\""){
            date = x.date.substring(1, date.length-1)
        }
        return <tr key={i}><td>{file}</td><td>{date}</td><td>{map}</td><td>
            <Link to={`/advancedStats?file=${file}&map=${map}`}>Stats</Link></td></tr>
    })
    return <>
        <div >
            <table style={{width:"100%"}}>
                <thead>
                    <tr>
                        <th>File Name</th>
                        <th>Date Uploaded</th>
                        <th>Map</th>
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