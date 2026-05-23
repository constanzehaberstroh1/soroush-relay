import { useState, useEffect } from 'react';
import {
  ThemeProvider,
  CssBaseline,
  Box,
  Container,
  Grid,
  Paper,
  Typography,
  TextField,
  Card,
  CardContent,
  IconButton,
  List,
  ListItem,
  ListItemText,
  LinearProgress,
  Divider,
} from '@mui/material';
import {
  Brightness4 as DarkModeIcon,
  Brightness7 as LightModeIcon,
  Save as SaveIcon,
  PowerSettingsNew as ShutdownIcon,
  People as UsersIcon,
  Dns as ServerIcon,
  Speed as PerformanceIcon,
  Assessment as StatsIcon,
  Dashboard as DashboardIcon,
  HistoryToggleOff as LogsIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { lightTheme, darkTheme } from './theme';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import {
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
  Link,
  Outlet,
} from '@tanstack/react-router';

// Atomic Imports
import { Button } from './components/atoms/Button';
import { Badge } from './components/atoms/Badge';

// ──────────────────────────────────────────────────────────────────────────────
// Interfaces & Types
// ──────────────────────────────────────────────────────────────────────────────
interface ConnectedClient {
  id: string;
  ip: string;
  phone: string;
  uptime: string;
  bandwidth: string;
  status: 'active' | 'stale';
}

interface LogEntry {
  timestamp: string;
  type: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

// ──────────────────────────────────────────────────────────────────────────────
// Global Core State / Context Simulation
// ──────────────────────────────────────────────────────────────────────────────
let globalCpuUsage = 12;
let globalMemUsage = 34;
let globalClients: ConnectedClient[] = [];
let globalServerPort = 8080;
let globalSocksHost = '127.0.0.1';
let globalSocksPort = 1080;
let globalBandwidthLimit = 100;

let globalLogs: LogEntry[] = [
  { timestamp: new Date().toLocaleTimeString(), type: 'info', message: 'Soroush exit node engine initialized.' },
  { timestamp: new Date().toLocaleTimeString(), type: 'success', message: 'WebRTC Signaling handler active on path /apiws' },
  { timestamp: new Date().toLocaleTimeString(), type: 'info', message: 'Ready to route decapsulated tunnel traffic to open Web' },
];
let globalPerfData: { time: string; cpu: number; mem: number }[] = [];
let listeners: (() => void)[] = [];

const subscribe = (cb: () => void) => {
  listeners.push(cb);
  return () => {
    listeners = listeners.filter((l) => l !== cb);
  };
};

const notify = () => {
  listeners.forEach((cb) => cb());
};

const addLog = (message: string, type: 'info' | 'success' | 'warn' | 'error' = 'info') => {
  const time = new Date().toLocaleTimeString();
  globalLogs = [{ timestamp: time, type, message }, ...globalLogs].slice(0, 100);
  notify();
};

const updateMetrics = (cpu: number, mem: number) => {
  globalCpuUsage = cpu;
  globalMemUsage = mem;
  notify();
};

// ──────────────────────────────────────────────────────────────────────────────
// Root Layout Component
// ──────────────────────────────────────────────────────────────────────────────
function RootLayout() {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    return localStorage.getItem('server-theme-mode') === 'dark';
  });

  const handleThemeToggle = () => {
    const nextMode = !darkMode;
    setDarkMode(nextMode);
    localStorage.setItem('server-theme-mode', nextMode ? 'dark' : 'light');
  };

  const theme = darkMode ? darkTheme : lightTheme;

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', bgcolor: 'background.default' }}>
        {/* Header App Bar */}
        <Paper elevation={0} sx={{ borderBottom: 1, borderColor: 'divider', py: 2.5, px: 4, borderRadius: 0 }}>
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Box display="flex" alignItems="center" gap={2}>
              <ServerIcon color="primary" sx={{ fontSize: 28 }} className="server-pulsing-glow" />
              <Typography variant="h6" color="text.primary" fontWeight={800} style={{ letterSpacing: '-0.01em' }}>
                SOROUSH RELAY
                <Box component="span" sx={{ color: 'primary.main', ml: 1, fontWeight: 400, fontSize: '0.85em' }}>
                  Server Admin Panel
                </Box>
              </Typography>
            </Box>
            
            <Box display="flex" alignItems="center" gap={2}>
              <Badge
                label="ENGINE: ACTIVE"
                customVariant="online"
              />
              <IconButton onClick={handleThemeToggle} color="inherit">
                {darkMode ? <LightModeIcon /> : <DarkModeIcon />}
              </IconButton>
            </Box>
          </Box>
        </Paper>

        {/* Dynamic Navigation Bar */}
        <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper', px: 4, py: 1 }}>
          <Box display="flex" gap={1}>
            <Link
              to="/"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <DashboardIcon fontSize="small" /> System Health
            </Link>
            <Link
              to="/config"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <SettingsIcon fontSize="small" /> Gateway Settings
            </Link>
            <Link
              to="/logs"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <LogsIcon fontSize="small" /> Signaling Logs
            </Link>
          </Box>
        </Box>

        {/* Main Content Router Outlet */}
        <Container maxWidth="xl" sx={{ mt: 4, mb: 4, flexGrow: 1 }}>
          <Outlet />
        </Container>
      </Box>
    </ThemeProvider>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 1: Dashboard View
// ──────────────────────────────────────────────────────────────────────────────
function DashboardView() {
  const [cpuUsage, setCpuUsage] = useState(globalCpuUsage);
  const [memUsage, setMemUsage] = useState(globalMemUsage);
  const [perfData, setPerfData] = useState(globalPerfData);

  useEffect(() => {
    const unsub = subscribe(() => {
      setCpuUsage(globalCpuUsage);
      setMemUsage(globalMemUsage);
    });

    const interval = setInterval(() => {
      const nextCpu = Math.max(5, Math.min(95, globalCpuUsage + Math.floor(Math.random() * 11) - 5));
      const nextMem = Math.max(20, Math.min(80, globalMemUsage + Math.floor(Math.random() * 5) - 2));
      updateMetrics(nextCpu, nextMem);

      const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
      globalPerfData = [...globalPerfData, { time, cpu: nextCpu, mem: nextMem }].slice(-15);
      setPerfData(globalPerfData);
    }, 1500);

    return () => {
      unsub();
      clearInterval(interval);
    };
  }, []);

  return (
    <Grid container spacing={3}>
      {/* Left Column: CPU & Connected Users */}
      <Grid item xs={12} md={4}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
                  <Typography variant="h6" fontWeight={700}>
                    Engine Utilization
                  </Typography>
                  <PerformanceIcon color="primary" />
                </Box>
                
                <Box display="flex" flexDirection="column" gap={2}>
                  <Box>
                    <Box display="flex" justifyContent="space-between" mb={0.5}>
                      <Typography variant="body2" color="text.secondary">CPU Load</Typography>
                      <Typography variant="body2" fontWeight="bold">{cpuUsage}%</Typography>
                    </Box>
                    <LinearProgress variant="determinate" value={cpuUsage} color={cpuUsage > 80 ? "error" : "primary"} />
                  </Box>

                  <Box>
                    <Box display="flex" justifyContent="space-between" mb={0.5}>
                      <Typography variant="body2" color="text.secondary">Memory Usage</Typography>
                      <Typography variant="body2" fontWeight="bold">{memUsage}%</Typography>
                    </Box>
                    <LinearProgress variant="determinate" value={memUsage} color={memUsage > 85 ? "error" : "secondary"} />
                  </Box>
                  
                  <Divider sx={{ my: 1 }} />
                  
                  <Grid container spacing={1}>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">Uptime</Typography>
                      <Typography variant="body2" fontWeight="bold">6d 18h 22m</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="caption" color="text.secondary">Active Tunnels</Typography>
                      <Typography variant="body2" fontWeight="bold" color="primary.main">{globalClients.length} Channels</Typography>
                    </Grid>
                  </Grid>
                </Box>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
                  <Typography variant="h6" fontWeight={700}>
                    Active Client Pipelines
                  </Typography>
                  <UsersIcon color="secondary" />
                </Box>
                <Typography variant="caption" color="text.secondary">
                  Decapsulating incoming WebRTC data streams.
                </Typography>

                <Divider sx={{ my: 2 }} />

                <List dense>
                  {globalClients.length > 0 ? (
                    globalClients.map((client) => (
                      <ListItem key={client.id} sx={{ px: 0 }}>
                        <ListItemText
                          primary={client.phone}
                          secondary={
                            <Box component="span" display="flex" flexDirection="column">
                              <span>Endpoint IP: {client.ip}</span>
                              <Box component="span" display="flex" gap={1.5} alignItems="center" mt={0.5}>
                                <Badge
                                  label={client.uptime}
                                  customVariant="online"
                                />
                                <Typography variant="caption" color="text.secondary">
                                  Rate: {client.bandwidth}
                                </Typography>
                              </Box>
                            </Box>
                          }
                        />
                      </ListItem>
                    ))
                  ) : (
                    <Box py={2} textAlign="center">
                      <Typography variant="body2" color="text.secondary">
                        No active client pipelines linked.
                      </Typography>
                    </Box>
                  )}
                </List>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Grid>

      {/* Right Column: Graphs */}
      <Grid item xs={12} md={8}>
        <Card>
          <CardContent sx={{ height: 460, display: 'flex', flexDirection: 'column' }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6" fontWeight={700}>
                System Utilization Timeline
              </Typography>
              <StatsIcon color="primary" />
            </Box>
            
            <Box sx={{ flexGrow: 1, minHeight: 0 }}>
              {perfData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={perfData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.06)" />
                    <XAxis dataKey="time" stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <YAxis domain={[0, 100]} stroke="#475569" style={{ fontSize: '0.75rem' }} />
                    <Tooltip contentStyle={{ backgroundColor: '#0f172a', borderColor: '#334155' }} />
                    <Line type="monotone" dataKey="cpu" stroke="#a78bfa" strokeWidth={2} dot={false} name="CPU (%)" />
                    <Line type="monotone" dataKey="mem" stroke="#22d3ee" strokeWidth={2} dot={false} name="Memory (%)" />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <Box display="flex" alignItems="center" justifyContent="center" height="100%">
                  <Typography variant="body2" color="text.secondary">
                    Awaiting core metrics timeline...
                  </Typography>
                </Box>
              )}
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 2: Gateway Configuration View
// ──────────────────────────────────────────────────────────────────────────────
function ConfigView() {
  const [serverPort, setServerPort] = useState<number>(globalServerPort);
  const [socksHost, setSocksHost] = useState<string>(globalSocksHost);
  const [socksPort, setSocksPort] = useState<number>(globalSocksPort);
  const [bandwidthLimit, setBandwidthLimit] = useState<number>(globalBandwidthLimit);

  const handleSaveConfig = () => {
    globalServerPort = serverPort;
    globalSocksHost = socksHost;
    globalSocksPort = socksPort;
    globalBandwidthLimit = bandwidthLimit;
    addLog(`Configuration saved. Server Port: ${serverPort}, SOCKS5 Target: ${socksHost}:${socksPort}`, 'success');
  };

  const handleRestartServer = () => {
    addLog('Initiating graceful restart of Soroush relay engine...', 'warn');
    setTimeout(() => {
      addLog('Terminating existing WebRTC pipelines...', 'info');
      addLog('Re-binding WebSocket signaling service to :8080/apiws', 'success');
      addLog('Relay server successfully restarted.', 'success');
    }, 1500);
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" fontWeight={700} mb={2}>
          Engine Gateway Settings
        </Typography>
        
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6}>
            <TextField
              label="HTTP Gateway Listen Port"
              type="number"
              value={serverPort}
              onChange={(e) => setServerPort(Number(e.target.value))}
              fullWidth
              size="small"
            />
          </Grid>
          <Grid item xs={12} sm={6}>
            <TextField
              label="Rate Limit per WebRTC channel (MB/s)"
              type="number"
              value={bandwidthLimit}
              onChange={(e) => setBandwidthLimit(Number(e.target.value))}
              fullWidth
              size="small"
            />
          </Grid>
          <Grid item xs={12} sm={8}>
            <TextField
              label="Forward Target Host (SOCKS5/HTTP Proxy Target)"
              value={socksHost}
              onChange={(e) => setSocksHost(e.target.value)}
              fullWidth
              size="small"
            />
          </Grid>
          <Grid item xs={12} sm={4}>
            <TextField
              label="Target Port"
              type="number"
              value={socksPort}
              onChange={(e) => setSocksPort(Number(e.target.value))}
              fullWidth
              size="small"
            />
          </Grid>

          <Grid item xs={12}>
            <Box display="flex" gap={2} mt={1}>
              <Button
                customVariant="primary"
                startIcon={<SaveIcon />}
                onClick={handleSaveConfig}
              >
                Apply Settings
              </Button>
              <Button
                customVariant="danger"
                startIcon={<ShutdownIcon />}
                onClick={handleRestartServer}
              >
                Restart Engine
              </Button>
            </Box>
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 3: Logs View
// ──────────────────────────────────────────────────────────────────────────────
function LogsView() {
  const [logs, setLogs] = useState(globalLogs);

  useEffect(() => {
    return subscribe(() => {
      setLogs(globalLogs);
    });
  }, []);

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" fontWeight={700} mb={2}>
          Signaling & P2P Routing Logs
        </Typography>
        <Box
          sx={{
            height: 480,
            bgcolor: '#050212',
            borderRadius: 3,
            p: 3,
            overflowY: 'auto',
            fontFamily: 'Courier New, monospace',
            fontSize: '0.85rem',
            display: 'flex',
            flexDirection: 'column',
            gap: 1,
            boxShadow: 'inset 0 2px 10px rgba(0,0,0,0.8)',
          }}
        >
          {logs.map((log, index) => (
            <Box key={index} display="flex" gap={1.5}>
              <Box sx={{ color: '#64748b', userSelect: 'none' }}>
                [{log.timestamp}]
              </Box>
              <Box
                sx={{
                  color:
                    log.type === 'success'
                      ? '#10b981'
                      : log.type === 'warn'
                      ? '#f59e0b'
                      : log.type === 'error'
                      ? '#ef4444'
                      : '#94a3b8',
                }}
              >
                {log.message}
              </Box>
            </Box>
          ))}
        </Box>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// TanStack Router Definition (TypeScript Compliant Code Route Tree)
// ──────────────────────────────────────────────────────────────────────────────
const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardView,
});

const configRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/config',
  component: ConfigView,
});

const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/logs',
  component: LogsView,
});

const routeTree = rootRoute.addChildren([indexRoute, configRoute, logsRoute]);

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// App Component Mounting Provider
// ──────────────────────────────────────────────────────────────────────────────
export function App() {
  return <RouterProvider router={router} />;
}

export default App;
