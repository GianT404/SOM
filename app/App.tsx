import React from 'react';
import { StatusBar } from 'react-native';
import { PlayerProvider } from './src/contexts/PlayerContext';
import AppNavigator from './src/navigation/AppNavigator';
import { COLORS } from './src/theme';

export default function App() {
  return (
    <PlayerProvider>
      <StatusBar barStyle="light-content" backgroundColor={COLORS.background} />
      <AppNavigator />
    </PlayerProvider>
  );
}
