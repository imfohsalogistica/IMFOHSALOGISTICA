GRANT USAGE ON SCHEMA public TO imfohsa_app;

-- Tablas de consulta sincronizadas desde el estado privado de la aplicación.

-- Los cambios operativos se realizan en la aplicación, que sigue siendo la fuente de datos.

CREATE TABLE public.centros (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 nombre text GENERATED ALWAYS AS (datos->>'name') STORED,
 latitud numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'lat')='number' THEN (datos->>'lat')::numeric ELSE NULL END) STORED,
 longitud numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'lng')='number' THEN (datos->>'lng')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.centros ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.centros FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.centros TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.centros FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.centros IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.clientes (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 nombre text GENERATED ALWAYS AS (datos->>'name') STORED,
 telefono text GENERATED ALWAYS AS (datos->>'phone') STORED,
 direccion text GENERATED ALWAYS AS (datos->>'address') STORED,
 codigo_odoo text GENERATED ALWAYS AS (datos->>'odoo') STORED,
 latitud numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'lat')='number' THEN (datos->>'lat')::numeric ELSE NULL END) STORED,
 longitud numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'lng')='number' THEN (datos->>'lng')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.clientes ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.clientes FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.clientes TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.clientes FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.clientes IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.mensajeros (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 nombre text GENERATED ALWAYS AS (datos->>'name') STORED,
 telefono text GENERATED ALWAYS AS (datos->>'phone') STORED,
 centro_id text GENERATED ALWAYS AS (datos->>'centerId') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED,
 vehiculo text GENERATED ALWAYS AS (datos->>'vehicle') STORED,
 placa text GENERATED ALWAYS AS (datos->>'plate') STORED,
 tipo text GENERATED ALWAYS AS (datos->>'type') STORED
);

ALTER TABLE public.mensajeros ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.mensajeros FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.mensajeros TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.mensajeros FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.mensajeros IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.vehiculos (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 placa text GENERATED ALWAYS AS (datos->>'plate') STORED,
 marca text GENERATED ALWAYS AS (datos->>'brand') STORED,
 modelo text GENERATED ALWAYS AS (datos->>'model') STORED,
 tipo text GENERATED ALWAYS AS (datos->>'type') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED,
 asignado_a text GENERATED ALWAYS AS (datos->>'assignedTo') STORED
);

ALTER TABLE public.vehiculos ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.vehiculos FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.vehiculos TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.vehiculos FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.vehiculos IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.pedidos (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 numero_pedido text GENERATED ALWAYS AS (datos->>'orderNo') STORED,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 cliente_id text GENERATED ALWAYS AS (datos->>'clientId') STORED,
 cliente text GENERATED ALWAYS AS (datos->>'clientName') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 centro_id text GENERATED ALWAYS AS (datos->>'centerId') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED,
 tipo_operacion text GENERATED ALWAYS AS (datos->>'operationType') STORED,
 estado_pago text GENERATED ALWAYS AS (datos->>'paymentStatus') STORED,
 numero_recibo text GENERATED ALWAYS AS (datos->>'receiptNo') STORED,
 monto numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'amount')='number' THEN (datos->>'amount')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.pedidos ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.pedidos FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.pedidos TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.pedidos FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.pedidos IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.combustible (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 vehiculo text GENERATED ALWAYS AS (datos->>'vehicle') STORED,
 monto numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'amount')='number' THEN (datos->>'amount')::numeric ELSE NULL END) STORED,
 galones numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'gallons')='number' THEN (datos->>'gallons')::numeric ELSE NULL END) STORED,
 kilometros numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'kmReal')='number' THEN (datos->>'kmReal')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.combustible ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.combustible FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.combustible TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.combustible FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.combustible IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.costos_manuales (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 categoria text GENERATED ALWAYS AS (datos->>'category') STORED,
 monto numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'amount')='number' THEN (datos->>'amount')::numeric ELSE NULL END) STORED,
 pedido text GENERATED ALWAYS AS (datos->>'so') STORED
);

ALTER TABLE public.costos_manuales ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.costos_manuales FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.costos_manuales TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.costos_manuales FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.costos_manuales IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.inventario_movimientos (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 pedido_id text GENERATED ALWAYS AS (datos->>'orderId') STORED,
 producto text GENERATED ALWAYS AS (datos->>'product') STORED,
 ubicacion text GENERATED ALWAYS AS (datos->>'location') STORED,
 bodega_origen text GENERATED ALWAYS AS (datos->>'sourceWarehouse') STORED,
 cantidad numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'quantity')='number' THEN (datos->>'quantity')::numeric ELSE NULL END) STORED,
 existencia numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'existence')='number' THEN (datos->>'existence')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.inventario_movimientos ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.inventario_movimientos FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.inventario_movimientos TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.inventario_movimientos FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.inventario_movimientos IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.traslados_bodega (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 pedido_id text GENERATED ALWAYS AS (datos->>'orderId') STORED,
 origen text GENERATED ALWAYS AS (datos->>'from') STORED,
 destino text GENERATED ALWAYS AS (datos->>'to') STORED,
 cantidad numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'quantity')='number' THEN (datos->>'quantity')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.traslados_bodega ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.traslados_bodega FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.traslados_bodega TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.traslados_bodega FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.traslados_bodega IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.documentos_despacho (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 archivo text GENERATED ALWAYS AS (datos->>'fileName') STORED,
 url text GENERATED ALWAYS AS (datos->>'url') STORED,
 estado_extraccion text GENERATED ALWAYS AS (datos->>'extractionStatus') STORED,
 capturado_en text GENERATED ALWAYS AS (datos->>'capturedAt') STORED,
 origen text GENERATED ALWAYS AS (datos->>'source') STORED
);

ALTER TABLE public.documentos_despacho ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.documentos_despacho FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.documentos_despacho TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.documentos_despacho FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.documentos_despacho IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.solicitudes_panel (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 solicitante text GENERATED ALWAYS AS (datos->>'requester') STORED,
 area text GENERATED ALWAYS AS (datos->>'area') STORED,
 centro_costo text GENERATED ALWAYS AS (datos->>'costCenter') STORED,
 destino text GENERATED ALWAYS AS (datos->>'destination') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED
);

ALTER TABLE public.solicitudes_panel ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.solicitudes_panel FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.solicitudes_panel TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.solicitudes_panel FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.solicitudes_panel IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.asignaciones_panel (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 vehiculo_id text GENERATED ALWAYS AS (datos->>'vehicleId') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 destino text GENERATED ALWAYS AS (datos->>'destination') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED
);

ALTER TABLE public.asignaciones_panel ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.asignaciones_panel FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.asignaciones_panel TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.asignaciones_panel FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.asignaciones_panel IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.viajes_panel (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 vehiculo_id text GENERATED ALWAYS AS (datos->>'vehicleId') STORED,
 pedido text GENERATED ALWAYS AS (datos->>'so') STORED,
 kilometros numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'kmReal')='number' THEN (datos->>'kmReal')::numeric ELSE NULL END) STORED,
 monto_factura numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'invoiceValue')='number' THEN (datos->>'invoiceValue')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.viajes_panel ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.viajes_panel FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.viajes_panel TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.viajes_panel FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.viajes_panel IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.historial_rutas (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 pedido_id text GENERATED ALWAYS AS (datos->>'orderId') STORED,
 tipo text GENERATED ALWAYS AS (datos->>'type') STORED,
 registrado_en text GENERATED ALWAYS AS (datos->>'at') STORED
);

ALTER TABLE public.historial_rutas ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.historial_rutas FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.historial_rutas TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.historial_rutas FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.historial_rutas IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.historial_cobros (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 pedido_id text GENERATED ALWAYS AS (datos->>'orderId') STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED,
 monto numeric GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'amount')='number' THEN (datos->>'amount')::numeric ELSE NULL END) STORED
);

ALTER TABLE public.historial_cobros ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.historial_cobros FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.historial_cobros TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.historial_cobros FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.historial_cobros IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.actividad (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 tipo text GENERATED ALWAYS AS (datos->>'type') STORED,
 usuario text GENERATED ALWAYS AS (datos->>'user') STORED,
 registrado_en text GENERATED ALWAYS AS (datos->>'at') STORED
);

ALTER TABLE public.actividad ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.actividad FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.actividad TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.actividad FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.actividad IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.reportes_mejora (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 titulo text GENERATED ALWAYS AS (datos->>'title') STORED,
 estado text GENERATED ALWAYS AS (datos->>'status') STORED
);

ALTER TABLE public.reportes_mejora ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.reportes_mejora FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.reportes_mejora TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.reportes_mejora FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.reportes_mejora IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.indicadores_sistema (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 fecha text GENERATED ALWAYS AS (datos->>'date') STORED,
 tipo text GENERATED ALWAYS AS (datos->>'type') STORED,
 titulo text GENERATED ALWAYS AS (datos->>'title') STORED,
 mensaje text GENERATED ALWAYS AS (datos->>'message') STORED
);

ALTER TABLE public.indicadores_sistema ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.indicadores_sistema FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.indicadores_sistema TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.indicadores_sistema FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.indicadores_sistema IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.disponibilidad_diaria (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE public.disponibilidad_diaria ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.disponibilidad_diaria FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.disponibilidad_diaria TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.disponibilidad_diaria FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.disponibilidad_diaria IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.resumenes_diarios (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE public.resumenes_diarios ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.resumenes_diarios FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.resumenes_diarios TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.resumenes_diarios FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.resumenes_diarios IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.configuracion (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE public.configuracion ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.configuracion FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.configuracion TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.configuracion FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.configuracion IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE TABLE public.usuarios (
 id text PRIMARY KEY,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 posicion_origen bigint NOT NULL,
 version_origen bigint NOT NULL,
 actualizado_en timestamptz NOT NULL DEFAULT now() ,
 usuario text GENERATED ALWAYS AS (datos->>'username') STORED,
 nombre text GENERATED ALWAYS AS (datos->>'name') STORED,
 rol text GENERATED ALWAYS AS (datos->>'role') STORED,
 activo boolean GENERATED ALWAYS AS (CASE WHEN jsonb_typeof(datos->'active')='boolean' THEN (datos->>'active')::boolean ELSE NULL END) STORED,
 mensajero_id text GENERATED ALWAYS AS (datos->>'messengerId') STORED
);

ALTER TABLE public.usuarios ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON public.usuarios FROM PUBLIC, anon, authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.usuarios TO imfohsa_app;

CREATE POLICY imfohsa_backend ON public.usuarios FOR ALL TO imfohsa_app USING(true) WITH CHECK(true);

COMMENT ON TABLE public.usuarios IS 'IMFOHSA: tabla sincronizada desde la aplicación. Editar datos desde la aplicación; sincronización en una sola dirección.';

CREATE INDEX pedidos_fecha_idx ON public.pedidos(fecha);

CREATE INDEX pedidos_cliente_id_idx ON public.pedidos(cliente_id);

CREATE INDEX pedidos_mensajero_id_idx ON public.pedidos(mensajero_id);

CREATE INDEX pedidos_estado_idx ON public.pedidos(estado);

CREATE INDEX inventario_movimientos_pedido_id_idx ON public.inventario_movimientos(pedido_id);

CREATE INDEX combustible_mensajero_id_idx ON public.combustible(mensajero_id);

CREATE INDEX asignaciones_panel_vehiculo_id_idx ON public.asignaciones_panel(vehiculo_id);

CREATE INDEX historial_rutas_mensajero_id_idx ON public.historial_rutas(mensajero_id);

CREATE FUNCTION imfohsa_private.sync_business_tables(p_kind text, p_payload jsonb, p_version bigint, p_previous jsonb DEFAULT NULL)
RETURNS void LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog AS $fn$
DECLARE
 mapping jsonb := '{"centers":["centros","array"],"clients":["clientes","array"],"messengers":["mensajeros","array"],"vehicles":["vehiculos","array"],"orders":["pedidos","array"],"fuel":["combustible","array"],"manualCosts":["costos_manuales","array"],"inventoryLedger":["inventario_movimientos","array"],"warehouseMovements":["traslados_bodega","array"],"dispatchDocuments":["documentos_despacho","array"],"panelRequests":["solicitudes_panel","array"],"panelAssignments":["asignaciones_panel","array"],"panelTrips":["viajes_panel","array"],"routeLearning":["historial_rutas","array"],"creditsLog":["historial_cobros","array"],"activityLog":["actividad","array"],"improvementReports":["reportes_mejora","array"],"systemInsights":["indicadores_sistema","array"],"dailyAvailability":["disponibilidad_diaria","object"],"dailySnapshots":["resumenes_diarios","object"],"settings":["configuracion","singleton"]}'::jsonb;
 entry record;
 source_key text;
 target_table text;
 shape text;
 items jsonb;
 raw_value jsonb;
 id_field text;
BEGIN
 IF p_kind = 'admins' THEN
  mapping := '{"admins":["usuarios","array"]}'::jsonb;
 ELSIF p_kind <> 'state' THEN
  RAISE EXCEPTION 'Documento no permitido';
 END IF;
 FOR entry IN SELECT key,value FROM jsonb_each(mapping) LOOP
  source_key := entry.key;
  target_table := entry.value->>0;
  shape := entry.value->>1;
  IF p_previous IS NOT NULL AND
    (CASE WHEN p_kind='admins' THEN p_payload IS NOT DISTINCT FROM p_previous ELSE p_payload->source_key IS NOT DISTINCT FROM p_previous->source_key END) THEN
   CONTINUE;
  END IF;
  raw_value := CASE WHEN p_kind='admins' THEN p_payload ELSE p_payload->source_key END;
  IF shape='array' THEN
   items := CASE WHEN jsonb_typeof(raw_value)='array' THEN raw_value ELSE '[]'::jsonb END;
  ELSIF shape='object' THEN
   SELECT coalesce(jsonb_agg(jsonb_build_object('id',key,'_value',value)),'[]'::jsonb) INTO items
    FROM jsonb_each(CASE WHEN jsonb_typeof(raw_value)='object' THEN raw_value ELSE '{}'::jsonb END);
  ELSE
   items := CASE WHEN jsonb_typeof(raw_value)='object' THEN jsonb_build_array(jsonb_build_object('id','general','_value',raw_value)) ELSE '[]'::jsonb END;
  END IF;
  id_field := CASE WHEN p_kind='admins' THEN 'username' ELSE 'id' END;
  -- Identifiers come only from the fixed mapping above; document values are parameters.
  EXECUTE format($sql$
   INSERT INTO public.%I(id,datos,posicion_origen,version_origen)
   SELECT coalesce(nullif(item->> $3,''),'@row:'||ordinality::text),
    (CASE WHEN $4='array' THEN item ELSE item->'_value' END)
      - ARRAY['passwordHash','password','password_hash','googleMapsApiKey','token','routeAccessToken','routeAccessDevice','panelRequestPublicToken','clientUploadPublicToken','sellerPublicToken'],
    ordinality,$2
   FROM jsonb_array_elements($1) WITH ORDINALITY a(item,ordinality)
   ON CONFLICT(id) DO UPDATE SET datos=EXCLUDED.datos,posicion_origen=EXCLUDED.posicion_origen,version_origen=EXCLUDED.version_origen,actualizado_en=now()
  $sql$,target_table) USING items,p_version,id_field,shape;
  EXECUTE format($sql$
   DELETE FROM public.%I target WHERE NOT EXISTS (
    SELECT 1 FROM jsonb_array_elements($1) WITH ORDINALITY a(item,ordinality)
    WHERE target.id=coalesce(nullif(item->>$2,''),'@row:'||ordinality::text)
   )
  $sql$,target_table) USING items,id_field;
 END LOOP;
END;
$fn$;

REVOKE ALL ON FUNCTION imfohsa_private.sync_business_tables(text,jsonb,bigint,jsonb) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION imfohsa_private.sync_business_tables(text,jsonb,bigint,jsonb) TO imfohsa_app;
CREATE FUNCTION imfohsa_private.business_tables_trigger()
RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog AS $fn$
BEGIN
 PERFORM imfohsa_private.sync_business_tables(NEW.key,NEW.payload,NEW.version,CASE WHEN TG_OP='UPDATE' THEN OLD.payload ELSE NULL END);
 RETURN NEW;
END;
$fn$;
REVOKE ALL ON FUNCTION imfohsa_private.business_tables_trigger() FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION imfohsa_private.business_tables_trigger() TO imfohsa_app;
CREATE TRIGGER sync_imfohsa_business_tables AFTER INSERT OR UPDATE OF payload,version ON imfohsa_private.documents
FOR EACH ROW EXECUTE FUNCTION imfohsa_private.business_tables_trigger();
-- Source and mirror writes commit atomically; migration locks prevent stale initial copies.
LOCK TABLE imfohsa_private.documents IN SHARE ROW EXCLUSIVE MODE;
SELECT imfohsa_private.sync_business_tables(key,payload,version,NULL) FROM imfohsa_private.documents;
