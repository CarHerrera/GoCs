import DemoTable from './components/demoTables.tsx'
import PlayerPage from './components/PlayerPage.tsx';
import { useId } from 'react';
import { BrowserRouter, Link, Routes, Route } from 'react-router-dom';
import TestKonva from './testingKonva.tsx';
import Login from './components/Login.tsx';
import TeamStatsDashboard from './components/TeamPage.tsx';
import { AuthProvider } from './context/AuthContext';
import Navbar from './components/Navbar';
import SeasonPage from './components/SeasonMatches.tsx';
import MatchPage from './components/MatchPage.tsx';
import Whiteboard from './components/Whiteboard.tsx';
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
      <AuthProvider>
        <Navbar></Navbar>  
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/demoList" element={<DemoTable />} />
              <Route path="/MatchPage" element={<MatchPage  />} />
              <Route path="/accountHome" element={
                <PlayerPage ></PlayerPage>}/>
              <Route path="/Test" element={<TestKonva  />} />
              <Route path ="/Whiteboard" element={<Whiteboard></Whiteboard>}></Route>
              <Route path="/Team" element={<TeamStatsDashboard></TeamStatsDashboard>}></Route>
              <Route path ="/season/:id" element={<SeasonPage></SeasonPage>}/>
            </Routes>
        </AuthProvider>
      </BrowserRouter>
    </>
  )
}

export default App
