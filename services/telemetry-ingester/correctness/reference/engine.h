#pragma once

struct Fill {
    double price;
    double quantity;
    const char* side;
};

Fill match_order(double price, double quantity, const char* side);
