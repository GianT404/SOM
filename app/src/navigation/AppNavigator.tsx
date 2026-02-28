import React from 'react';
import { View, StyleSheet } from 'react-native';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, RADIUS, RETRO_BORDER } from '../theme';

import HomeScreen from '../screens/HomeScreen';
import SearchScreen from '../screens/SearchScreen';
import PlaylistScreen from '../screens/PlaylistScreen';
import SettingsScreen from '../screens/SettingsScreen';
import NowPlayingScreen from '../screens/NowPlayingScreen';
import LyricsScreen from '../screens/LyricsScreen';

const Stack = createNativeStackNavigator();

const AppNavigator = () => {
    return (
        <NavigationContainer>
            <View style={{ flex: 1, backgroundColor: COLORS.background }}>
                <Stack.Navigator screenOptions={{ headerShown: false }}>
                    <Stack.Screen name="Home" component={HomeScreen} />
                    <Stack.Screen name="Search" component={SearchScreen} />
                    <Stack.Screen name="Playlist" component={PlaylistScreen} />
                    <Stack.Screen name="Settings" component={SettingsScreen} />
                    <Stack.Screen name="NowPlaying" component={NowPlayingScreen} options={{ presentation: 'modal', animation: 'slide_from_bottom' }} />
                    <Stack.Screen name="Lyrics" component={LyricsScreen} options={{ animation: 'slide_from_right' }} />
                </Stack.Navigator>
            </View>
        </NavigationContainer>
    );
};

export default AppNavigator;
