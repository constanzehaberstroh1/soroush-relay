import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#0d9488', // Soroush teal-blue
      light: '#2dd4bf',
      dark: '#115e59',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#6366f1', // Vibrant indigo
      light: '#818cf8',
      dark: '#4f46e5',
    },
    background: {
      default: '#f8fafc', // Slate 50
      paper: 'rgba(255, 255, 255, 0.7)', // Frosted paper
    },
    text: {
      primary: '#0f172a', // Slate 900
      secondary: '#475569', // Slate 600
    },
    divider: 'rgba(15, 23, 42, 0.08)',
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
    h1: { fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontWeight: 700, letterSpacing: '-0.01em' },
    h3: { fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    button: { fontWeight: 600, textTransform: 'none' },
  },
  shape: {
    borderRadius: 16,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          background: 'rgba(255, 255, 255, 0.7) !important',
          backdropFilter: 'blur(16px) saturate(120%)',
          border: '1px solid rgba(15, 23, 42, 0.08)',
          borderRadius: 16,
          boxShadow: '0 8px 32px rgba(31, 38, 135, 0.05)',
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
      },
    },
  },
});

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#2dd4bf', // Sleek Soroush neon-teal
      light: '#99f6e4',
      dark: '#0f766e',
      contrastText: '#0f172a',
    },
    secondary: {
      main: '#818cf8', // Cyber indigo
      light: '#a5b4fc',
      dark: '#4f46e5',
    },
    background: {
      default: '#020617', // Space dark
      paper: 'rgba(15, 23, 42, 0.45)', // Translucent glassmorphic slate
    },
    text: {
      primary: '#f8fafc', // Slate 50
      secondary: '#94a3b8', // Slate 400
    },
    divider: 'rgba(255, 255, 255, 0.08)',
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
    h1: { fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontWeight: 700, letterSpacing: '-0.01em' },
    h3: { fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    button: { fontWeight: 600, textTransform: 'none' },
  },
  shape: {
    borderRadius: 16,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          background: 'rgba(15, 23, 42, 0.45) !important',
          backdropFilter: 'blur(16px) saturate(120%)',
          border: '1px solid rgba(255, 255, 255, 0.07)',
          borderRadius: 16,
          boxShadow: '0 10px 30px -10px rgba(0, 0, 0, 0.5)',
          transition: 'all 0.4s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            borderColor: 'rgba(13, 148, 136, 0.3)',
            boxShadow: '0 15px 35px -5px rgba(13, 148, 136, 0.05), 0 0 15px rgba(13, 148, 136, 0.05)',
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
      },
    },
  },
});
