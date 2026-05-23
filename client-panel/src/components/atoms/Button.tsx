import React from 'react';
import { Button as MuiButton } from '@mui/material';
import type { ButtonProps as MuiButtonProps } from '@mui/material';
import { styled, keyframes } from '@mui/material/styles';

export interface AtomicButtonProps extends Omit<MuiButtonProps, 'variant'> {
  customVariant?: 'primary' | 'secondary' | 'danger' | 'success' | 'glow';
  glowColor?: string;
}

const glowAnimation = (color: string) => keyframes`
  0% {
    box-shadow: 0 0 0 0 ${color}bf;
  }
  70% {
    box-shadow: 0 0 0 10px ${color}00;
  }
  100% {
    box-shadow: 0 0 0 0 ${color}00;
  }
`;

const StyledButton = styled(MuiButton, {
  shouldForwardProp: (prop) => prop !== 'customVariant' && prop !== 'glowColor',
})<AtomicButtonProps>(({ theme, customVariant = 'primary', glowColor }) => {
  const isDark = theme.palette.mode === 'dark';
  
  const baseStyles = {
    borderRadius: 12,
    fontWeight: 600,
    textTransform: 'none' as const,
    padding: '8px 20px',
    transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
    fontFamily: '"Inter", sans-serif',
  };

  switch (customVariant) {
    case 'primary':
      return {
        ...baseStyles,
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        '&:hover': {
          backgroundColor: theme.palette.primary.dark,
          transform: 'translateY(-2px)',
          boxShadow: isDark
            ? '0 8px 24px rgba(45, 212, 191, 0.3)'
            : '0 8px 20px rgba(13, 148, 136, 0.2)',
        },
        '&:active': {
          transform: 'translateY(0)',
        },
      };

    case 'secondary':
      return {
        ...baseStyles,
        backgroundColor: isDark ? 'rgba(255, 255, 255, 0.03)' : 'rgba(15, 23, 42, 0.02)',
        border: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.08)' : 'rgba(15, 23, 42, 0.08)'}`,
        color: theme.palette.text.primary,
        backdropFilter: 'blur(8px)',
        '&:hover': {
          backgroundColor: isDark ? 'rgba(255, 255, 255, 0.07)' : 'rgba(15, 23, 42, 0.05)',
          borderColor: isDark ? 'rgba(255, 255, 255, 0.2)' : 'rgba(15, 23, 42, 0.2)',
          transform: 'translateY(-1px)',
        },
      };

    case 'danger':
      return {
        ...baseStyles,
        backgroundColor: isDark ? '#ef4444' : '#dc2626',
        color: '#ffffff',
        '&:hover': {
          backgroundColor: isDark ? '#b91c1c' : '#991b1b',
          transform: 'translateY(-2px)',
          boxShadow: '0 8px 20px rgba(239, 68, 68, 0.25)',
        },
      };

    case 'success':
      return {
        ...baseStyles,
        backgroundColor: isDark ? '#10b981' : '#059669',
        color: '#ffffff',
        '&:hover': {
          backgroundColor: isDark ? '#047857' : '#065f46',
          transform: 'translateY(-2px)',
          boxShadow: '0 8px 20px rgba(16, 185, 129, 0.25)',
        },
      };

    case 'glow':
      const defaultGlowColor = theme.palette.primary.main;
      const targetGlow = glowColor || defaultGlowColor;
      return {
        ...baseStyles,
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        animation: `${glowAnimation(targetGlow)} 2.5s infinite`,
        '&:hover': {
          backgroundColor: theme.palette.primary.dark,
          transform: 'scale(1.03)',
        },
      };
      
    default:
      return baseStyles;
  }
});

export const Button: React.FC<AtomicButtonProps> = ({ children, ...props }) => {
  return (
    <StyledButton {...props}>
      {children}
    </StyledButton>
  );
};

export default Button;
