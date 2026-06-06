#include "engine.h"

Fill match_order(double price, double quantity, const char* side) {
    Fill fill{price, quantity, side};
    return fill;
}
