import DemoTable from './components/demoTables.tsx'
import AdvancedStats from './components/statInfo.tsx';
import PlayerPage from './components/PlayerPage.tsx';
import { useId } from 'react';
import { BrowserRouter, Link, Routes, Route } from 'react-router-dom';
import TestKonva from './testingKonva.tsx';
import Login from './components/Login.tsx';
function Home() {
  const fileId = useId();
  return <>
      <div>
        <Login></Login>
      </div>
      {/* <div>
        <h1>Or Check out these already parsed demos</h1>
        <Link to="/demoList"><button>Click Me!</button></Link>
      </div> */}
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
          <Route path="/accountHome" element={
            <PlayerPage ></PlayerPage>}/>
          <Route path="/Test" element={<TestKonva  />} />
        </Routes>
    </BrowserRouter>
      
    </>
  )
}

export default App
