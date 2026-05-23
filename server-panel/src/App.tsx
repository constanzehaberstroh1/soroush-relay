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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Avatar,
  CircularProgress,
  Alert,
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
  LockOutlined as LockIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  AccountCircle as AccountIcon,
  Refresh as RefreshIcon,
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

interface SoroushAccount {
  id: string;
  phoneNumber: string;
  name: string;
  soroushUserId: string;
  sessionToken: string;
  status: 'connected' | 'error' | 'idle';
  lastActive: string;
  createdAt: string;
}

interface LogEntry {
  timestamp: string;
  type: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

// REST Backend Service Utility
const API_BASE = '/api';

const getHeaders = () => {
  const token = localStorage.getItem('server-admin-token');
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
};

// ──────────────────────────────────────────────────────────────────────────────
// Root Layout Component (Wraps dashboard after authentication)
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

  const handleLogout = () => {
    localStorage.removeItem('server-admin-token');
    window.location.reload();
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
                  Server Exit Node Panel
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
              <Button customVariant="secondary" size="small" onClick={handleLogout} style={{ padding: '4px 10px', fontSize: '0.75rem' }}>
                Logout
              </Button>
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
              to="/accounts"
              activeProps={{ style: { color: theme.palette.primary.main, fontWeight: 700, backgroundColor: 'rgba(124, 58, 237, 0.08)' } }}
              style={{ textDecoration: 'none', color: theme.palette.text.secondary, padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: 8, fontSize: '0.85rem', fontWeight: 500 }}
            >
              <AccountIcon fontSize="small" /> Soroush exit Accounts
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
// Page 1: Dashboard View (Live polled backend system metrics)
// ──────────────────────────────────────────────────────────────────────────────
function DashboardView() {
  const [cpuUsage, setCpuUsage] = useState<number>(12);
  const [memUsage, setMemUsage] = useState<number>(34);
  const [activeClientsCount, setActiveClientsCount] = useState<number>(0);
  const [connectedClients, setConnectedClients] = useState<ConnectedClient[]>([]);
  const [perfData, setPerfData] = useState<{ time: string; cpu: number; mem: number }[]>([]);

  const fetchStats = async () => {
    try {
      const res = await fetch(`${API_BASE}/stats`, { headers: getHeaders() });
      if (res.status === 401) {
        localStorage.removeItem('server-admin-token');
        window.location.reload();
        return;
      }
      if (res.ok) {
        const data = await res.json();
        setCpuUsage(data.cpu);
        setMemUsage(data.memory);
        setActiveClientsCount(data.activeTunnels);

        const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        setPerfData((prev) => [...prev, { time, cpu: data.cpu, mem: data.memory }].slice(-15));
      }
    } catch (err) {
      console.error('Failed fetching stats:', err);
    }
  };

  const fetchClients = async () => {
    try {
      const res = await fetch(`${API_BASE}/clients`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setConnectedClients(data);
      }
    } catch (err) {
      console.error('Failed fetching pipelines:', err);
    }
  };

  useEffect(() => {
    fetchStats();
    fetchClients();
    const interval = setInterval(() => {
      fetchStats();
      fetchClients();
    }, 2000);

    return () => clearInterval(interval);
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
                      <Typography variant="body2" fontWeight="bold" color="primary.main">{activeClientsCount} Channels</Typography>
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
                  {connectedClients.length > 0 ? (
                    connectedClients.map((client) => (
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
// Page 2: Soroush Exit Accounts Manager View
// ──────────────────────────────────────────────────────────────────────────────
function AccountsView() {
  const [accounts, setAccounts] = useState<SoroushAccount[]>([]);
  const [openAddDialog, setOpenAddDialog] = useState<boolean>(false);
  const [newPhone, setNewPhone] = useState<string>('');
  const [newName, setNewName] = useState<string>('');
  const [verifyCode, setVerifyCode] = useState<string>('');
  const [step, setStep] = useState<1 | 2>(1);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [helperNotice, setHelperNotice] = useState<string>('');

  const fetchAccounts = async () => {
    try {
      const res = await fetch(`${API_BASE}/accounts`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setAccounts(data);
      }
    } catch (err) {
      console.error('Failed fetching accounts:', err);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  const handleRequestPIN = async () => {
    if (!newPhone || !newName) {
      setErrorMsg('Please specify both Friendly Name and Soroush Phone number!');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/accounts/request-otp`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ phoneNumber: newPhone, name: newName }),
      });
      const data = await res.json();
      if (res.ok) {
        setStep(2);
        setHelperNotice('OTP request successfully initiated! Check the system logs terminal or the "Signaling Logs" tab to retrieve your PIN code.');
      } else {
        setErrorMsg(data.error || 'Failed to dispatch verification SMS.');
      }
    } catch (err) {
      setErrorMsg('Error contacting Soroush signaling gateway.');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyPIN = async () => {
    if (!verifyCode) {
      setErrorMsg('Please enter the 5-digit verification PIN.');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/accounts/verify-otp`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ phoneNumber: newPhone, name: newName, code: verifyCode }),
      });
      const data = await res.json();
      if (res.ok) {
        setOpenAddDialog(false);
        setNewPhone('');
        setNewName('');
        setVerifyCode('');
        setStep(1);
        setHelperNotice('');
        fetchAccounts();
      } else {
        setErrorMsg(data.error || 'Incorrect PIN code entered.');
      }
    } catch (err) {
      setErrorMsg('Error verifying PIN on Soroush gateway.');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteAccount = async (id: string) => {
    if (!confirm('Are you sure you want to completely delete this Soroush credential?')) return;
    try {
      const res = await fetch(`${API_BASE}/accounts?id=${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      });
      if (res.ok) {
        fetchAccounts();
      }
    } catch (err) {
      console.error('Delete account request error:', err);
    }
  };

  return (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
          <Typography variant="h6" fontWeight={700}>
            Registered Soroush Exit Credentials
          </Typography>
          <Button customVariant="primary" startIcon={<AddIcon />} onClick={() => setOpenAddDialog(true)}>
            Add Account
          </Button>
        </Box>
        <Typography variant="body2" color="text.secondary" mb={3}>
          Link exit node messaging accounts to negotiate WebRTC signaling handshakes with inbound clients.
        </Typography>

        <Divider sx={{ mb: 3 }} />

        {accounts.length > 0 ? (
          <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 3, overflow: 'hidden', bgcolor: 'transparent' }}>
            <Table>
              <TableHead sx={{ bgcolor: 'rgba(255,255,255,0.02)' }}>
                <TableRow>
                  <TableCell style={{ fontWeight: 700 }}>Account Name</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Phone Number</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Soroush ID</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Session Token</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Status</TableCell>
                  <TableCell style={{ fontWeight: 700 }}>Registered</TableCell>
                  <TableCell style={{ fontWeight: 700 }} align="center">Action</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {accounts.map((acc) => (
                  <TableRow key={acc.id} hover sx={{ '&:hover': { bgcolor: 'rgba(124, 58, 237, 0.04) !important' } }}>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={1.5}>
                        <Avatar sx={{ bgcolor: 'rgba(124, 58, 237, 0.1)', color: 'secondary.main', fontSize: '0.85rem', fontWeight: 'bold' }}>
                          {acc.name.substring(0, 2).toUpperCase()}
                        </Avatar>
                        <Box>
                          <Typography variant="body2" fontWeight={600}>{acc.name}</Typography>
                          <Typography variant="caption" color="text.secondary">ID: {acc.id.substring(0, 8)}</Typography>
                        </Box>
                      </Box>
                    </TableCell>
                    <TableCell style={{ fontFamily: 'monospace' }}>{acc.phoneNumber}</TableCell>
                    <TableCell style={{ fontFamily: 'monospace' }}>{acc.soroushUserId}</TableCell>
                    <TableCell style={{ fontFamily: 'monospace' }}>
                      {acc.sessionToken ? `${acc.sessionToken.substring(0, 14)}...` : 'N/A'}
                    </TableCell>
                    <TableCell>
                      <Badge
                        label={acc.status.toUpperCase()}
                        customVariant={acc.status === 'connected' ? 'online' : 'idle'}
                      />
                    </TableCell>
                    <TableCell style={{ fontSize: '0.75rem' }}>
                      {new Date(acc.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell align="center">
                      <IconButton onClick={() => handleDeleteAccount(acc.id)} color="error" size="small">
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : (
          <Box py={6} textAlign="center">
            <AccountIcon sx={{ fontSize: 48, opacity: 0.15, mb: 1, color: 'text.secondary' }} />
            <Typography variant="body2" color="text.secondary">
              No registered Soroush exit credentials found. Link credentials to begin.
            </Typography>
          </Box>
        )}

        {/* Add Account Dialog Wizard */}
        <Dialog open={openAddDialog} onClose={() => { setOpenAddDialog(false); setErrorMsg(''); setStep(1); setHelperNotice(''); }} fullWidth maxWidth="xs">
          <DialogTitle fontWeight={800} style={{ paddingBottom: '8px' }}>
            Add Soroush Exit Account
          </DialogTitle>
          <DialogContent>
            {errorMsg && <Alert severity="error" sx={{ mb: 2, borderRadius: 2 }}>{errorMsg}</Alert>}
            {helperNotice && <Alert severity="success" sx={{ mb: 2, borderRadius: 2 }}>{helperNotice}</Alert>}

            {step === 1 ? (
              <Box display="flex" flexDirection="column" gap={2.5} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Link details through the Soroush authentication framework.
                </Typography>
                <TextField
                  label="Friendly Name"
                  placeholder="e.g. Server Exit Account"
                  fullWidth
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  size="small"
                  disabled={loading}
                />
                <TextField
                  label="Soroush Phone"
                  placeholder="+989123456789"
                  fullWidth
                  value={newPhone}
                  onChange={(e) => setNewPhone(e.target.value)}
                  size="small"
                  disabled={loading}
                />
              </Box>
            ) : (
              <Box display="flex" flexDirection="column" gap={2.5} mt={1}>
                <Typography variant="body2" color="text.secondary">
                  Enter the 5-digit verification PIN received on your Soroush mobile app.
                </Typography>
                <TextField
                  label="SMS Pin code"
                  placeholder="12345"
                  fullWidth
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value)}
                  size="small"
                  disabled={loading}
                />
              </Box>
            )}
          </DialogContent>
          <DialogActions sx={{ p: 3, pt: 1 }}>
            <Button customVariant="secondary" onClick={() => { setOpenAddDialog(false); setErrorMsg(''); setStep(1); setHelperNotice(''); }} disabled={loading}>
              Cancel
            </Button>
            {step === 1 ? (
              <Button customVariant="primary" onClick={handleRequestPIN} disabled={loading}>
                {loading ? <CircularProgress size={20} color="inherit" /> : "Request Pin"}
              </Button>
            ) : (
              <Button customVariant="glow" onClick={handleVerifyPIN} disabled={loading}>
                {loading ? <CircularProgress size={20} color="inherit" /> : "Verify Pin"}
              </Button>
            )}
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Page 3: Gateway Configuration View
// ──────────────────────────────────────────────────────────────────────────────
function ConfigView() {
  const [serverPort, setServerPort] = useState<number>(8080);
  const [socksHost, setSocksHost] = useState<string>('127.0.0.1');
  const [socksPort, setSocksPort] = useState<number>(1080);
  const [bandwidthLimit, setBandwidthLimit] = useState<number>(100);

  const fetchConfig = async () => {
    try {
      const res = await fetch(`${API_BASE}/config`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setServerPort(data.serverPort);
        setSocksHost(data.socksHost);
        setSocksPort(data.socksPort);
        setBandwidthLimit(data.bandwidthLimit);
      }
    } catch (err) {
      console.error('Failed fetching server config:', err);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  const handleSaveConfig = async () => {
    try {
      const res = await fetch(`${API_BASE}/config`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          serverPort: Number(serverPort),
          socksHost,
          socksPort: Number(socksPort),
          bandwidthLimit: Number(bandwidthLimit),
        }),
      });
      if (res.ok) {
        alert('Server gateway settings updated successfully!');
      }
    } catch (err) {
      console.error('Failed saving server config:', err);
    }
  };

  const handleRestartServer = () => {
    alert('Graceful restart trace dispatched to signaling coordinator.');
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
// Page 4: Logs View (Live polled backend signaling logs)
// ──────────────────────────────────────────────────────────────────────────────
function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const fetchLogs = async () => {
    try {
      const res = await fetch(`${API_BASE}/logs`, { headers: getHeaders() });
      if (res.ok) {
        const data = await res.json();
        setLogs(data);
      }
    } catch (err) {
      console.error('Failed fetching signaling logs:', err);
    }
  };

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 1500);

    return () => clearInterval(interval);
  }, []);

  return (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
          <Typography variant="h6" fontWeight={700}>
            Signaling & P2P Routing Logs
          </Typography>
          <IconButton onClick={fetchLogs} color="primary" size="small">
            <RefreshIcon fontSize="small" />
          </IconButton>
        </Box>
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
          {logs.length > 0 ? (
            logs.map((log, index) => (
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
            ))
          ) : (
            <Box py={4} textAlign="center" color="text.secondary">
              Awaiting logs buffer payload...
            </Box>
          )}
        </Box>
      </CardContent>
    </Card>
  );
}

// ──────────────────────────────────────────────────────────────────────────────
// Elegant Glassmorphic Login Screen
// ──────────────────────────────────────────────────────────────────────────────
interface LoginScreenProps {
  onLoginSuccess: (token: string) => void;
}

function LoginScreen({ onLoginSuccess }: LoginScreenProps) {
  const [username, setUsername] = useState<string>('');
  const [password, setPassword] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setErrorMsg('Username and password fields are both required!');
      return;
    }
    setErrorMsg('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (res.ok && data.token) {
        localStorage.setItem('server-admin-token', data.token);
        onLoginSuccess(data.token);
      } else {
        setErrorMsg(data.error || 'Authentication rejected. Check credentials.');
      }
    } catch (err) {
      setErrorMsg('Connection error. Server may be offline.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'radial-gradient(circle at top, #0f172a, #020617)',
        px: 2,
      }}
    >
      <Card
        sx={{
          maxWidth: 400,
          width: '100%',
          p: 3,
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
          background: 'rgba(15, 23, 42, 0.45) !important',
          backdropFilter: 'blur(20px)',
          border: '1px solid rgba(255, 255, 255, 0.08)',
          borderRadius: 4,
        }}
      >
        <CardContent>
          <Box display="flex" flexDirection="column" alignItems="center" mb={4}>
            <Avatar sx={{ m: 1, bgcolor: 'primary.main', width: 56, height: 56, boxShadow: '0 0 20px rgba(139, 92, 246, 0.4)' }}>
              <LockIcon style={{ color: '#0f172a', fontSize: 28 }} />
            </Avatar>
            <Typography variant="h5" component="h1" fontWeight={800} mt={1} color="text.primary" textAlign="center">
              SOROUSH RELAY
            </Typography>
            <Typography variant="caption" color="text.secondary" textAlign="center">
              Server Administrator Dashboard Portal
            </Typography>
          </Box>

          {errorMsg && (
            <Alert severity="error" sx={{ mb: 3, borderRadius: 2 }}>
              {errorMsg}
            </Alert>
          )}

          <Box component="form" onSubmit={handleLoginSubmit} display="flex" flexDirection="column" gap={3}>
            <TextField
              label="Username"
              variant="outlined"
              fullWidth
              size="medium"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={loading}
              InputLabelProps={{ style: { color: 'rgba(255,255,255,0.7)' } }}
              sx={{ input: { color: '#ffffff' } }}
            />
            <TextField
              label="Password"
              type="password"
              variant="outlined"
              fullWidth
              size="medium"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
              InputLabelProps={{ style: { color: 'rgba(255,255,255,0.7)' } }}
              sx={{ input: { color: '#ffffff' } }}
            />
            <Button
              type="submit"
              customVariant="glow"
              fullWidth
              disabled={loading}
              style={{ paddingTop: '12px', paddingBottom: '12px', fontSize: '1rem', fontWeight: 700 }}
            >
              {loading ? <CircularProgress size={24} color="inherit" /> : "Authenticate Login"}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
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

const accountsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/accounts',
  component: AccountsView,
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

const routeTree = rootRoute.addChildren([indexRoute, accountsRoute, configRoute, logsRoute]);

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// App Component Mounting Provider with JWT auth wrapper
// ──────────────────────────────────────────────────────────────────────────────
export function App() {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('server-admin-token');
  });

  const [checking, setChecking] = useState<boolean>(true);

  // Perform lightweight validation on startup
  useEffect(() => {
    const validateLocalToken = async () => {
      if (!token) {
        setChecking(false);
        return;
      }
      try {
        const res = await fetch(`${API_BASE}/admin/me`, {
          headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) {
          localStorage.removeItem('server-admin-token');
          setToken(null);
        }
      } catch (err) {
        console.error('Authentication gateway offline:', err);
      } finally {
        setChecking(false);
      }
    };
    validateLocalToken();
  }, [token]);

  if (checking) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" minHeight="100vh" bgcolor="#020617">
        <CircularProgress color="primary" />
      </Box>
    );
  }

  if (!token) {
    return (
      <ThemeProvider theme={darkTheme}>
        <CssBaseline />
        <LoginScreen onLoginSuccess={(tok) => setToken(tok)} />
      </ThemeProvider>
    );
  }

  return <RouterProvider router={router} />;
}

export default App;
