# CONOCIMIENTO CONSOLIDADO — LOGÍSTICA IMFOHSA
Fecha: 2026-08-25
Base: R7.7.7

## Jerarquía
R7.7.7 es la base técnica. Las correcciones y aprendizajes de “Revisión Del Planificador Rutas” se aplican encima. No restaurar versiones que eliminen funciones aprobadas. Supabase = persistencia central; Vercel = publicación web.

## Módulos
Dashboard; Planificador de Rutas; Pedidos y Despachos; Clientes Odoo; Mensajeros; Créditos y Cobros; Consulta Vendedores; Flotilla; Solicitud de Vehículo; Combustibles y Costos; Configuración; Inventarios.

## Prioridad actual
Dashboard, Planificador, Pedidos/Despachos, Mensajero móvil, Clientes Odoo y Consulta Vendedores. Clientes Odoo será base maestra de clientes.

## Casilla física
Leer, guardar y mostrar de forma destacada en Pedidos/Despachos, Planificador y Mensajero. Sirve para identificación física, visibilidad Bodega→Mensajero→Cliente y evitar cruces.

## Lector de picking
Extraer SO/Gestión, cliente, dirección completa, teléfono, monto si existe, observaciones/instrucciones, vendedor y plazo de pago obligatorio. Dirección manuscrita clara junto al bloque de dirección tiene prioridad. No confundir casilla aislada con cantidades. Detectar SO duplicadas, adjuntar/unificar/agregar a CEDI, consolidación y restricciones AM/PM.

## Origen e inventario
BZ4/PICK = Bodega Zona 4; B-REP = Bodega Zona 4; BUSAC = USAC Zona 12; CDZ16 = Zona 16. Extraer producto, ubicación, cantidad, existencia, desde y a. Analizar centralización y costo de recolección.

## Pago/cobro
Plazo no determina por sí solo cobro. NEOLINK o depósito/transferencia validado = pagado/no cobrar. 120 días sin evidencia = crédito/no cobrar. Institucional IGSS: Logística solo entrega.

## Rutas
Promesa capital 3h; Express 2h. Cortes 08:00, 09:30, 12:30, 14:00. AM/PM separados. Salida 5–10 min tras asignación. Espera cliente alertas 5/10/15 min. Reorden manual exacto y optimización.

## Mensajero móvil
Ruta AM/PM, SO, cliente, dirección, casilla, navegación, llegada, espera, atención, entregado/no entregado, intentos, fotos/evidencias y odómetro inicio/fin. Tercera visita no efectiva: anular según regla. Devolución con motivo y responsable.

## Roles
Gerente General/Jefe Logística: todo. Asistentes: Dashboard, Planificador, Pedidos/Despachos, Mensajeros + descarga. Ventas: vista limitada y Consulta Vendedores. NLIMA modifica Z16 no Z4; GOLIVA modifica Z4 no Z16.

## KPI
Entregas, entregados/no entregados, km, eficiencia AM/PM, ranking, promesa 3h, express, devoluciones, puntualidad, tiempo, costo/entrega, km real vs estimado, costo/km, gestiones/cobros.

## Parámetros
125cc=96 km/gal; 150cc=88 km/gal; depreciación Q0.37/km real. Editables en Configuración.

## Migración empresarial
Conservar fotos, evidencias, histórico, usuarios/roles, reglas, configuración, prompts operativos, skills y decisiones documentadas. El conocimiento operativo debe quedar persistido y versionado, no depender solo del chat.
