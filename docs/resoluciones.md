# Resoluciones: original vs GX

## Qué hace el original (`Game.exe` EP5.4, Ghidra)

`0x408240` fija el modo `DAT_007198dc` según el tamaño de ventana:

| Modo | Resolución | Aspecto de proyección |
| --- | --- | --- |
| 0 | 800x600 | 1.25 forzado |
| 1 | 1024x768 y **cualquier tamaño no listado** | 1.25 forzado |
| 2 | 1280x1024 | 1.25 forzado |
| 3 | 1360x768 | real |
| 4 | 1920x1200 | real |
| 5 | 1680x1050 | real |
| 6 | 1280x720 | real |
| 7 | 1600x900 | real |

Solo unas pocas UI dependen del modo:

- **Barra inferior** (`0x483610`): modo 0 usa `main_bottom8` (800 px); modos 1, 3 y 6
  usan la de 1024 px; modos 2, 4, 5 y 7 usan los `*12` (arte a doble densidad, ~1250 px
  de contenido, ~1.22x), colocada en `alto-0x82`. Sus botones, barra de EXP y bendición
  cambian de posición con la variante.
- **Pantallas de carga** (`0x4b7b40`): imagen a tamaño de pantalla (GX ya la estira).
- **Layout guardado** (`0x526870`/`0x526c30`): una sección `INTERFACE_WxH` por
  resolución con las posiciones de las ventanas (GX guarda centros relativos, así que ya
  es independiente de la resolución).
- Login, selección y chat (`0x4b8ff0`, `0x4ba650`, `0x4bd680`, `0x4f8c40`, `0x4f9000`,
  `0x4fcd70`) también consultan el modo; pendientes de revisar.

El resto de ventanas conserva su tamaño en píxeles a cualquier resolución.

## Qué hace GX

- Proyección: `projectionAspect` replica el 1.25 forzado de 800x600, 1024x768 y 1280x1024.
- Barra inferior: **un solo arte (1024) que solo se reduce**. `ui.UIScale(w,h)` =
  `min(1, w/1024, h/768)` (mínimo 0.4): a 1024x768 o más grande queda como siempre,
  sin ampliarse; en ventanas menores se encoge en proporción (800x600 → 0.78, el mismo
  tamaño que el arte `main_bottom8` del original, que ya no se usa). Botones, barra de
  EXP, bendición y zonas de ratón escalan juntos, y las barras de acción, pociones, cursor
  y menú contextual se anclan con `ui.BottomBarHeight` en lugar del 64 fijo.
- **Toda la UI** (pantallas previas, HUD, ventanas, chat) con la misma `UIScale`: a menos de
  1024x768 la UI trabaja en una ventana virtual de ese tamaño y se dibuja reducida
  (`Sprites.Begin` con el tamaño virtual); el ratón se convierte dividiendo por la escala
  (`App.Update`). La proyección de etiquetas y la selección 3D usan coordenadas
  normalizadas, así que siguen coherentes. Lo que depende de píxeles reales se convierte:
  la copia de profundidad de nombres (`App.physW/physH`, `pickNameVisibleAt`), los
  viewports 3D dentro de ventanas (`render.SetUIScale` + `InViewport`) y el cursor
  (`CursorIndex`). El desplazamiento del ratón capturado (cámara) no se escala.
- A 1024x768 o más grande no cambia nada.
