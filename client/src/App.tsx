import DemoTable from './demoTables.tsx'
import AdvancedStats from './statInfo';
import { useId } from 'react';
import { BrowserRouter, Link, Routes, Route } from 'react-router-dom';
function Home() {
  const fileId = useId();
  return <>
  
  <div>
        <form>
          <label htmlFor={fileId}>Select a File:</label>
          <input type="file" accept=".dem" id={fileId}></input>
          <input type="submit"></input>
        </form>
      </div>
      <div>
        <h1>Or Check out these already parsed demos</h1>
        <Link to="/demoList"><button>Click Me!</button></Link>
      </div>
  </>
}
function App() {
  

  return (
    <>

    <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/demoList" element={<DemoTable />} />
          <Route path="/advancedStats" element={<AdvancedStats  />} />
        </Routes>
    </BrowserRouter>
      
    </>
  )
}

export default App
