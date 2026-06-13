#ifndef ENGINE_H
#define ENGINE_H

#include <string>
#include <map>
#include <deque>
#include <unordered_map>
#include <unordered_set>
#include <cstdint>

struct Order {
    std::string orderId;
    std::string orderType; // "LIMIT" | "MARKET" | "CANCEL"
    std::string side;      // "BUY" | "SELL" | ""
    double price;
    double quantity;
    uint64_t seqNum;
    double filledQuantity;
};

struct FillResult {
    double price;
    double quantity;
    std::string side;
};

class MatchingEngine {
public:
    MatchingEngine() : seqCounter(0) {}
    FillResult process(const Order& order);

private:
    uint64_t seqCounter;
    
    // Bids sorted descending by price
    std::map<double, std::deque<Order*>, std::greater<double>> bidBook;
    
    // Asks sorted ascending by price
    std::map<double, std::deque<Order*>> askBook;
    
    // Fast lookup for all resting orders (including partially filled)
    std::unordered_map<std::string, Order> orderLookup;
    
    // Track fully filled or cancelled orders
    std::unordered_set<std::string> inactiveOrders;

    FillResult processLimitBuy(Order& order);
    FillResult processLimitSell(Order& order);
    FillResult processMarketBuy(Order& order);
    FillResult processMarketSell(Order& order);
    FillResult processCancel(Order& order);
};

// Parser / serializer
Order parseOrder(const std::string& line);
std::string fillToJson(const FillResult& fill);

#endif // ENGINE_H
