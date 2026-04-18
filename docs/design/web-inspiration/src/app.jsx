// Main App component
const { Sidebar, TasksPage, NetworkPage, AutomationPage, BridgesPage, KnowledgePage, SkillsPage, SessionPage, SettingsPage } = window;

function App() {
  const [route, setRoute] = React.useState(() => localStorage.getItem('agh:route') || 'tasks');
  const [sessionId, setSessionId] = React.useState(() => localStorage.getItem('agh:session') || 'c-1');

  React.useEffect(() => { localStorage.setItem('agh:route', route); }, [route]);
  React.useEffect(() => { localStorage.setItem('agh:session', sessionId); }, [sessionId]);

  return (
    <div className="app">
      <Sidebar route={route} setRoute={setRoute} sessionId={sessionId} setSessionId={setSessionId} />
      <div className="main" data-screen-label={route}>
        {route === 'tasks' && <TasksPage />}
        {route === 'network' && <NetworkPage />}
        {route === 'automation' && <AutomationPage />}
        {route === 'bridges' && <BridgesPage />}
        {route === 'knowledge' && <KnowledgePage />}
        {route === 'skills' && <SkillsPage />}
        {route === 'session' && <SessionPage sessionId={sessionId} />}
        {route === 'settings' && <SettingsPage />}
      </div>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
