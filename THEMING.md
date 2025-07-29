# Kalo Theme System

Kalo supports a flexible theme system that allows you to customize the appearance of the application through JSON configuration files.

## Theme Configuration

Themes are defined in JSON files with a human-friendly structure:

```json
{
  "name": "My Custom Theme",
  "description": "A beautiful custom theme",
  "colors": {
    "primary": "75",
    "secondary": "39", 
    "background": "234",
    "surface": "235",
    "border": "240",
    "borderFocus": "75",
    "text": "255",
    "textMuted": "244",
    "textAccent": "39",
    "success": "34",
    "warning": "226",
    "error": "196",
    "cursor": "207"
  }
}
```

## Color Values

Colors use ANSI 256-color codes (0-255). Some common values:

- **Black/Dark**: 0, 16, 232-255 (grayscale)
- **White/Light**: 15, 255, 252-255 (light grayscale)
- **Red**: 1, 9, 196, 203
- **Green**: 2, 10, 34, 46
- **Yellow**: 3, 11, 220, 226
- **Blue**: 4, 12, 25, 39, 75
- **Purple**: 5, 13, 57, 99, 207
- **Cyan**: 6, 14, 51, 87

## Theme Locations

Themes are loaded from these locations in order:

1. `~/.kalo/themes/[name].json` (user themes)
2. `./themes/[name].json` (built-in themes)
3. Default theme (code fallback)

## Built-in Themes

- **default**: The standard Kalo theme
- **dark**: Modern dark theme with blue accents
- **light**: Clean light theme for bright environments

## Color Mapping

Theme colors are mapped to UI elements as follows:

- `primary`: Main accent color (borders, titles)
- `secondary`: Secondary accent (URLs, section headers)
- `background`: Main background color
- `surface`: Panel backgrounds
- `border`: Default border color
- `borderFocus`: Focused element borders
- `text`: Primary text color
- `textMuted`: Secondary/muted text
- `textAccent`: Accent text (section names)
- `success`: Success states (HTTP 200, etc.)
- `warning`: Warning states
- `error`: Error states
- `cursor`: Text cursor and selection color

## Creating Custom Themes

1. Create a new JSON file in `~/.kalo/themes/`
2. Define your color palette
3. Missing colors will fallback to defaults
4. Restart Kalo to load the new theme

## Theme Development Tips

- Use online ANSI color references to pick colors
- Test themes in both light and dark terminals
- Ensure sufficient contrast for readability
- Consider colorblind-friendly palettes

## Future Enhancements

- Runtime theme switching
- Theme validation
- Color name aliases
- RGB/Hex color support
- Theme editor UI