CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE etsy_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    etsy_user_id TEXT NOT NULL,
    shop_id BIGINT NOT NULL,
    shop_name TEXT NOT NULL,
    scope TEXT NOT NULL,
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT NOT NULL,
    access_token_expires_at TIMESTAMPTZ NOT NULL,
    connection_status TEXT NOT NULL DEFAULT 'connected',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ NULL,
    last_sync_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT etsy_connections_status_check
        CHECK (connection_status IN ('connected', 'disconnected', 'reconnect_required'))
);

CREATE UNIQUE INDEX etsy_connections_user_shop_active_unique_idx
    ON etsy_connections (user_id, shop_id)
    WHERE disconnected_at IS NULL;

CREATE INDEX etsy_connections_user_id_idx
    ON etsy_connections (user_id);

CREATE INDEX etsy_connections_shop_id_idx
    ON etsy_connections (shop_id);

CREATE INDEX etsy_connections_status_idx
    ON etsy_connections (connection_status);

CREATE TABLE etsy_oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    state TEXT NOT NULL,
    code_verifier_encrypted TEXT NOT NULL,
    redirect_after TEXT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX etsy_oauth_states_state_unique_idx
    ON etsy_oauth_states (state);

CREATE INDEX etsy_oauth_states_user_id_idx
    ON etsy_oauth_states (user_id);

CREATE INDEX etsy_oauth_states_expires_at_idx
    ON etsy_oauth_states (expires_at);

CREATE INDEX etsy_oauth_states_unused_idx
    ON etsy_oauth_states (state, expires_at)
    WHERE used_at IS NULL;

CREATE TABLE etsy_sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    connection_id UUID NOT NULL REFERENCES etsy_connections(id) ON DELETE CASCADE,
    shop_id BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    triggered_by TEXT NOT NULL DEFAULT 'manual',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    error_message TEXT NULL,
    listings_seen INT NOT NULL DEFAULT 0,
    listings_upserted INT NOT NULL DEFAULT 0,
    images_upserted INT NOT NULL DEFAULT 0,
    inventory_items_upserted INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT etsy_sync_runs_status_check
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT etsy_sync_runs_triggered_by_check
        CHECK (triggered_by IN ('manual', 'system')),
    CONSTRAINT etsy_sync_runs_counts_check
        CHECK (
            listings_seen >= 0
            AND listings_upserted >= 0
            AND images_upserted >= 0
            AND inventory_items_upserted >= 0
        )
);

CREATE INDEX etsy_sync_runs_user_id_idx
    ON etsy_sync_runs (user_id);

CREATE INDEX etsy_sync_runs_connection_id_idx
    ON etsy_sync_runs (connection_id);

CREATE INDEX etsy_sync_runs_shop_id_idx
    ON etsy_sync_runs (shop_id);

CREATE INDEX etsy_sync_runs_status_idx
    ON etsy_sync_runs (status);

CREATE INDEX etsy_sync_runs_started_at_idx
    ON etsy_sync_runs (started_at DESC);

CREATE UNIQUE INDEX etsy_sync_runs_one_running_per_shop_idx
    ON etsy_sync_runs (user_id, shop_id)
    WHERE status IN ('queued', 'running');

CREATE TABLE etsy_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    connection_id UUID NOT NULL REFERENCES etsy_connections(id) ON DELETE CASCADE,
    shop_id BIGINT NOT NULL,
    etsy_listing_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NULL,
    state TEXT NULL,
    url TEXT NULL,
    price_amount BIGINT NULL,
    price_divisor INT NULL,
    currency_code TEXT NULL,
    quantity INT NULL,
    sku TEXT NULL,
    is_digital BOOLEAN NULL,
    created_timestamp BIGINT NULL,
    updated_timestamp BIGINT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT etsy_listings_quantity_check
        CHECK (quantity IS NULL OR quantity >= 0),
    CONSTRAINT etsy_listings_price_amount_check
        CHECK (price_amount IS NULL OR price_amount >= 0),
    CONSTRAINT etsy_listings_price_divisor_check
        CHECK (price_divisor IS NULL OR price_divisor > 0)
);

CREATE UNIQUE INDEX etsy_listings_user_listing_unique_idx
    ON etsy_listings (user_id, etsy_listing_id);

CREATE INDEX etsy_listings_connection_id_idx
    ON etsy_listings (connection_id);

CREATE INDEX etsy_listings_shop_id_idx
    ON etsy_listings (shop_id);

CREATE INDEX etsy_listings_title_idx
    ON etsy_listings (title);

CREATE INDEX etsy_listings_state_idx
    ON etsy_listings (state);

CREATE INDEX etsy_listings_sku_idx
    ON etsy_listings (sku);

CREATE TABLE etsy_listing_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES etsy_listings(id) ON DELETE CASCADE,
    etsy_listing_id BIGINT NOT NULL,
    etsy_listing_image_id BIGINT NOT NULL,
    rank INT NULL,
    url_75x75 TEXT NULL,
    url_170x135 TEXT NULL,
    url_570xN TEXT NULL,
    url_fullxfull TEXT NULL,
    width INT NULL,
    height INT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT etsy_listing_images_rank_check
        CHECK (rank IS NULL OR rank >= 0),
    CONSTRAINT etsy_listing_images_width_check
        CHECK (width IS NULL OR width >= 0),
    CONSTRAINT etsy_listing_images_height_check
        CHECK (height IS NULL OR height >= 0)
);

CREATE UNIQUE INDEX etsy_listing_images_listing_image_unique_idx
    ON etsy_listing_images (listing_id, etsy_listing_image_id);

CREATE INDEX etsy_listing_images_listing_id_idx
    ON etsy_listing_images (listing_id);

CREATE INDEX etsy_listing_images_etsy_listing_id_idx
    ON etsy_listing_images (etsy_listing_id);

CREATE INDEX etsy_listing_images_primary_idx
    ON etsy_listing_images (listing_id)
    WHERE is_primary = true;

CREATE TABLE etsy_inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES etsy_listings(id) ON DELETE CASCADE,
    etsy_listing_id BIGINT NOT NULL,
    etsy_product_id BIGINT NULL,
    sku TEXT NULL,
    barcode TEXT NULL,
    quantity INT NOT NULL,
    price_amount BIGINT NULL,
    price_divisor INT NULL,
    currency_code TEXT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    property_values_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    variation_summary TEXT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT etsy_inventory_items_quantity_check
        CHECK (quantity >= 0),
    CONSTRAINT etsy_inventory_items_price_amount_check
        CHECK (price_amount IS NULL OR price_amount >= 0),
    CONSTRAINT etsy_inventory_items_price_divisor_check
        CHECK (price_divisor IS NULL OR price_divisor > 0),
    CONSTRAINT etsy_inventory_items_property_values_json_check
        CHECK (jsonb_typeof(property_values_json) = 'array')
);

CREATE UNIQUE INDEX etsy_inventory_items_listing_product_unique_idx
    ON etsy_inventory_items (listing_id, etsy_product_id)
    WHERE etsy_product_id IS NOT NULL;

CREATE INDEX etsy_inventory_items_listing_id_idx
    ON etsy_inventory_items (listing_id);

CREATE INDEX etsy_inventory_items_etsy_listing_id_idx
    ON etsy_inventory_items (etsy_listing_id);

CREATE INDEX etsy_inventory_items_sku_idx
    ON etsy_inventory_items (sku);

CREATE INDEX etsy_inventory_items_barcode_idx
    ON etsy_inventory_items (barcode);

CREATE INDEX etsy_inventory_items_quantity_idx
    ON etsy_inventory_items (quantity);

CREATE INDEX etsy_inventory_items_enabled_idx
    ON etsy_inventory_items (is_enabled);
