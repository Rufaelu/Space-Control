package com.spacecontrol.game;

import org.libsdl.app.SDLActivity;

/**
 * The app's only screen. SDL's activity does the work: it loads libSDL2.so and
 * the game (libmain.so, built from the Go code) and calls the game's SDL_main.
 */
public class SpaceControlActivity extends SDLActivity {
    @Override
    protected String[] getLibraries() {
        return new String[] { "SDL2", "main" };
    }
}
