import React, { useState, useEffect } from 'react';
import { BrowserRouter, Link, Routes, Route } from 'react-router-dom';

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
        return <tr key={i}><td>{x.filename}</td><td>{x.date.substring(0,11)}</td><td>{x.filesize}MB</td><td>{x.map}</td><td>
            <Link to={`/advancedStats?file=${x.filename}&map=${x.map}`}>Stats</Link></td></tr>
    })
    return <>
        <div >
            <table style={{width:"100%"}}>
                <thead>
                    <tr>
                        <th>File Name</th>
                        <th>Date Uploaded</th>
                        <th>filesize</th>
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