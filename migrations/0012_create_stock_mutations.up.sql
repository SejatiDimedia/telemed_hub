CREATE TABLE IF NOT EXISTS stock_mutations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medicine_id UUID NOT NULL REFERENCES medicines(id),
    mutation_type VARCHAR(10) NOT NULL CHECK (mutation_type IN ('in', 'out')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    stock_before INTEGER NOT NULL,
    stock_after INTEGER NOT NULL,
    reference_type VARCHAR(30) NOT NULL CHECK (reference_type IN ('initial_stock', 'manual_adjustment', 'order_fulfillment', 'order_cancel_refund')),
    reference_id UUID,
    notes TEXT DEFAULT '',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_mutations_medicine_id ON stock_mutations(medicine_id);
CREATE INDEX IF NOT EXISTS idx_stock_mutations_created_at ON stock_mutations(created_at DESC);
