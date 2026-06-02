-- Test seed data: one customer, one policy bundle.
INSERT INTO customers (id, name, tier)
VALUES ('11111111-1111-1111-1111-111111111111', 'Acme Networks', 'enterprise')
ON CONFLICT DO NOTHING;

INSERT INTO policies (customer_id, document, checksum)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '{"quorum":{"min_votes":3},"mitigation_templates":["flowspec_drop","bgp_blackhole"],"evidence_schema":"v1"}',
  'seedchecksum'
) ON CONFLICT (customer_id) DO NOTHING;
