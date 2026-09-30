# Skill oficial — Planificador de Rutas IMFOHSA R8

## Propósito
Esta Skill es la especificación maestra del módulo Planificador de Rutas. Cualquier cambio futuro debe conservar estas reglas antes de modificar interfaz, cálculo o flujo operativo.

## Principio principal
El sistema NO debe decidir primero el mensajero por cercanía en línea recta. La secuencia obligatoria es:
1. Validar pedido, centro, corte, ubicación y disponibilidad.
2. Construir matriz vial por calles entre CEDI y pedidos.
3. Aplicar restricciones duras.
4. Comparar alternativas completas.
5. Seleccionar mensajero + pedidos + orden de visita.
6. Validar la ruta final con Google Routes si está disponible; si falla, usar OSRM.
7. Mostrar la propuesta sin asignar.
8. Asignar únicamente después de confirmación humana.

## Fuente única de parámetros
Todos los parámetros se leen de Configuración. No se deben crear nuevos horarios o límites escritos directamente dentro del algoritmo.

Parámetros mínimos:
- routeWindows: corte, preparación, salida y retorno máximo.
- cuts: lista de cortes operativos.
- standardStopMinutes.
- trafficFactor.
- plannerUseMinAmountRule.
- plannerMinOrderAmount.
- plannerHardReturn.
- plannerHardAppointments.
- plannerLearningEnabled.
- plannerLearningMinSamples.
- plannerBalanceWeight.
- plannerZoneAffinityWeight.
- plannerCorrectionWeight.
- plannerMatrixBlockSize.
- googleMapsApiKey / googleRoutingPreference.

## Restricciones duras
Nunca forzar una asignación que viole una restricción marcada como dura.
- Mensajero ausente, en panel u ocupado: no elegible.
- Pedido anulado/finalizado: no elegible.
- Ubicación inválida: no entra al motor.
- Corte inexistente: no entra al motor.
- Retorno posterior al máximo: dejar pedido sin capacidad cuando plannerHardReturn=true.
- Cita/ventana de cliente incumplible: dejar pedido sin capacidad cuando plannerHardAppointments=true.
- El pedido no puede duplicarse ni desaparecer.
- Una ruta siempre debe incluir cálculo de salida desde el centro y retorno al centro.

## Regla por monto
La regla de monto es política configurable, no lógica fija. Si plannerUseMinAmountRule=true, pedidos menores a plannerMinOrderAmount quedan para asignación manual. Si está desactivada, todos los montos válidos pueden entrar al motor.

## Matriz vial
- Hasta 80 puntos: matriz vial en una consulta.
- Más de 80 puntos: dividir en bloques para soportar 100+ pedidos.
- Fuente preferida de asignación: OSRM por calles.
- Si la matriz vial no está disponible: respaldo geográfico explícitamente etiquetado.
- Nunca presentar respaldo geográfico como tráfico real.

## Validación final y tráfico
Prioridad:
1. Google Routes con tráfico cuando API Key y servicio funcionen.
2. OSRM por calles + factor configurable.
3. Respaldo geográfico/histórico solo si los anteriores fallan.

La fuente utilizada debe mostrarse al usuario.

## Aprendizaje operativo
El aprendizaje debe cambiar cálculos futuros, no solo guardar bitácora.
Aprender de:
- Tiempo real entre Llegué y cierre de gestión por cliente.
- Diferencia entre ETA inicial y hora real por corte.
- Reasignaciones manuales y afinidad cliente/mensajero.
- Historial suficiente según plannerLearningMinSamples.

No usar una muestra aislada como verdad. Cuando no hay suficientes datos, usar parámetros estándar.

## Citas y ventanas de cliente
Leer appointmentTime, appointment o promise cuando contengan horas. La secuencia debe incorporar espera si llega antes y marcar violación si llega después del límite. Con restricción dura activa, esa alternativa no es válida.

## Balance y asignación
La función objetivo debe combinar:
- incremento de tiempo vial;
- balance de carga semanal;
- afinidad geográfica/zona;
- aprendizaje de correcciones manuales;
- retorno dentro de ventana;
- citas del cliente.

Las restricciones duras tienen prioridad sobre cualquier puntaje.

## Confirmación
El botón inteligente debe analizar y proponer. No debe modificar messengerId mientras el usuario no presione CONFIRMAR ASIGNACIÓN.

## Reoptimización
La reoptimización de una ruta ya asignada debe respetar cortes y recalcular el orden por calles. Debe registrar el evento en routeLearning.

## QA obligatorio antes de liberar
Como mínimo validar:
- 50 pedidos.
- 120 pedidos.
- 4 cortes.
- 8 mensajeros.
- ausente y panel.
- direcciones inválidas.
- cita imposible.
- capacidad insuficiente.
- regla de monto activa/desactivada.
- Google disponible y Google fallando.
- OSRM fallando.
- ningún duplicado.
- ningún pedido perdido.
- retorno al CEDI.
- persistencia después de guardar/reiniciar.
- concurrencia/optimistic locking existente.

## Criterio de “100% funcional”
No significa garantizar que Google, Internet o tráfico nunca fallen. Significa que el módulo mantiene integridad, respeta reglas, explica la fuente de cálculo, degrada de forma segura y nunca fuerza una decisión inválida por la caída de un proveedor externo.

---
## Addendum R8.1 / R7.6.0 TEST — aprendizaje documental, inventario y operación

1. El corte operativo se infiere por hora de recepción y sigue siendo editable antes de guardar: 08:00, 09:30, 12:30 y 14:00. Después de 14:00 se propone 08:00 del siguiente día operativo (L-S).
2. El orden de parada es persistente mediante `routePosition` e independiente por corte. Drag & drop no debe mezclar cortes silenciosamente.
3. “Mostrar rutas” es dinámico: Todas, No asignadas, Solo asignadas, mensajero, Panel e Institucional. Se elimina la dependencia lógica de Ruta 1–4 fijas.
4. La propuesta inteligente debe usar matriz vial primero (`r8RoadMatrix`) y fallback explícito si el proveedor no está disponible.
5. Mensajeros: 10 totales; 8 Ruta, 1 Panel, 1 Institucional. Roles rotativos. Los 8 Ruta se distribuyen 4/4 en grupos A/B y los horarios 08:00/09:00 se intercambian semanalmente.
6. Express: SLA 120 minutos desde liberación de Logística; prioridad dinámica, no simplemente “poner primero”. No debe romper una ventana dura de otro cliente.
7. El lector conserva SO, cliente, dirección original/operativa, contacto, vendedor, plazo/estado/medio/referencia de pago, casilla física, instrucciones, acciones requeridas y SO relacionadas.
8. Origen de bodega: BZ4/PICK y B-REP = Zona 4; BUSAC = USAC Zona 12; CDZ16 = Zona 16. Conservar referencia interna.
9. Leer y guardar Producto, Ubicación, Cantidad, Existencia, Desde y A para análisis histórico de inventario.
10. El aprendizaje debe seguir el ciclo planificar → ejecutar → comparar → detectar error → identificar causa → aplicar corrección segura → medir mejora → informar resultado. Cambios de alto impacto requieren aprobación.
11. Informes deben separar KM productivos, internos y potencialmente evitables y relacionarlos con gasolina/depreciación para recomendar mejor ubicación de inventario.
