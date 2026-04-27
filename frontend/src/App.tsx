import { useEffect, useState } from "react";
import { Route, Switch, useLocation } from "wouter";
import { AuthProvider } from "./auth/AuthContext";
import { Layout } from "./components/Layout";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Login } from "./pages/Login";
import { Home } from "./pages/Home";
import { DeviceInfo } from "./pages/DeviceInfo";
import { Network } from "./pages/Network";
import { Settings } from "./pages/Settings";
import { SMS } from "./pages/SMS";
import { ATCmd } from "./pages/ATCmd";
import { Console } from "./pages/Console";
import { Account } from "./pages/Account";

export default function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <Switch>
          <Route path="/login"><Login /></Route>
          <Route>
            <ProtectedRoute>
              <Layout>
                <RoutedContent />
              </Layout>
            </ProtectedRoute>
          </Route>
        </Switch>
      </AuthProvider>
    </ErrorBoundary>
  );
}

// RoutedContent keeps the Console mounted across tab switches — its
// xterm terminal and websocket survive navigation so the user doesn't lose
// scrollback or get a fresh shell every time they leave the tab.
//
// We lazy-mount Console (don't open a websocket until the user first visits
// /console), then keep it alive forever, toggling visibility based on route.
function RoutedContent() {
  const [loc] = useLocation();
  const onConsole = loc === "/console";
  const [consoleMounted, setConsoleMounted] = useState(onConsole);
  useEffect(() => {
    if (onConsole) setConsoleMounted(true);
  }, [onConsole]);

  return (
    <>
      <div className={onConsole ? "hidden" : "h-full"}>
        <Switch>
          <Route path="/" component={Home} />
          <Route path="/deviceinfo" component={DeviceInfo} />
          <Route path="/network" component={Network} />
          <Route path="/scanner" component={Network} />
          <Route path="/settings" component={Settings} />
          <Route path="/sms" component={SMS} />
          <Route path="/atcmd" component={ATCmd} />
          <Route path="/account" component={Account} />
          <Route path="/console">{null /* handled by the persistent slot below */}</Route>
          <Route>Not found</Route>
        </Switch>
      </div>
      {consoleMounted && (
        <div className={onConsole ? "h-full" : "hidden"}>
          <Console />
        </div>
      )}
    </>
  );
}
