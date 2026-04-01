import { useEffect, useState } from "react";

function DemoPlayback({file}:{file:String}){
        const [stats, setStats] = useState<any>()
        console.log(file);
        useEffect(( ) => {
            let ignore = false;
            async function getStats(){
                return fetch(`http://localhost:4000/2DPlayback/${file}`, {
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
                    // console.log(data)
                    return data;
                })
            }
            async function getFetch(){
                const data = await getStats();
                if(!ignore){
                    setStats(data)
                    console.log(data)
                    console.log(stats)
                }
            }
            getFetch()
            return () => { ignore = true}
        }, [file])
    return <>Hello!</>
}

export default DemoPlayback;  