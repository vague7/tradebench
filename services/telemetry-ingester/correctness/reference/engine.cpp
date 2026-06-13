#include "engine.h"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <algorithm>

// Very basic JSON parser for flat, known schema
// {"orderId":"...","orderType":"...","side":"...","price":...,"quantity":...}
Order parseOrder(const std::string& line) {
    Order o;
    o.price = 0.0;
    o.quantity = 0.0;
    o.seqNum = 0;
    o.filledQuantity = 0.0;
    
    auto extractString = [&](const std::string& key) -> std::string {
        std::string search = "\"" + key + "\":\"";
        size_t pos = line.find(search);
        if (pos == std::string::npos) return "";
        pos += search.length();
        size_t end = line.find("\"", pos);
        if (end == std::string::npos) return "";
        return line.substr(pos, end - pos);
    };
    
    auto extractDouble = [&](const std::string& key) -> double {
        std::string search = "\"" + key + "\":";
        size_t pos = line.find(search);
        if (pos == std::string::npos) return 0.0;
        pos += search.length();
        size_t end1 = line.find(",", pos);
        size_t end2 = line.find("}", pos);
        size_t end = std::min(end1, end2);
        if (end == std::string::npos) return 0.0;
        try {
            return std::stod(line.substr(pos, end - pos));
        } catch (...) {
            return 0.0;
        }
    };

    o.orderId = extractString("orderId");
    o.orderType = extractString("orderType");
    o.side = extractString("side");
    o.price = extractDouble("price");
    o.quantity = extractDouble("quantity");

    return o;
}

std::string fillToJson(const FillResult& fill) {
    std::ostringstream oss;
    oss << "{\"price\":" << fill.price 
        << ",\"quantity\":" << fill.quantity 
        << ",\"side\":\"" << fill.side << "\"}";
    return oss.str();
}

FillResult MatchingEngine::process(const Order& incomingOrder) {
    Order o = incomingOrder;
    o.seqNum = ++seqCounter;
    
    if (o.orderType == "CANCEL") {
        return processCancel(o);
    }

    if (o.orderType == "MARKET") {
        if (o.side == "BUY") return processMarketBuy(o);
        if (o.side == "SELL") return processMarketSell(o);
        return {0.0, 0.0, ""};
    }

    if (o.orderType == "LIMIT") {
        if (o.side == "BUY") return processLimitBuy(o);
        if (o.side == "SELL") return processLimitSell(o);
        return {0.0, 0.0, ""};
    }

    return {0.0, 0.0, ""};
}

FillResult MatchingEngine::processCancel(Order& order) {
    if (inactiveOrders.count(order.orderId)) {
        return {0.0, 0.0, ""}; // Stale cancel
    }
    auto it = orderLookup.find(order.orderId);
    if (it == orderLookup.end()) {
        return {0.0, 0.0, ""}; // Unknown order
    }
    
    // Mark inactive and remove from books
    inactiveOrders.insert(order.orderId);
    
    Order& resting = it->second;
    if (resting.side == "BUY") {
        auto& q = bidBook[resting.price];
        for (auto qit = q.begin(); qit != q.end(); ++qit) {
            if ((*qit)->orderId == order.orderId && (*qit)->seqNum == resting.seqNum) {
                q.erase(qit);
                break;
            }
        }
    } else if (resting.side == "SELL") {
        auto& q = askBook[resting.price];
        for (auto qit = q.begin(); qit != q.end(); ++qit) {
            if ((*qit)->orderId == order.orderId && (*qit)->seqNum == resting.seqNum) {
                q.erase(qit);
                break;
            }
        }
    }
    
    return {0.0, 0.0, ""};
}

FillResult MatchingEngine::processLimitBuy(Order& order) {
    double totalFillQty = 0.0;
    double totalCost = 0.0;
    
    auto it = askBook.begin();
    while (it != askBook.end() && order.quantity > 0 && it->first <= order.price) {
        auto& q = it->second;
        while (!q.empty() && order.quantity > 0) {
            Order* resting = q.front();
            if (inactiveOrders.count(resting->orderId)) {
                q.pop_front();
                continue;
            }
            
            double matchQty = std::min(order.quantity, resting->quantity - resting->filledQuantity);
            
            totalFillQty += matchQty;
            totalCost += matchQty * resting->price;
            order.quantity -= matchQty;
            resting->filledQuantity += matchQty;
            
            if (resting->filledQuantity >= resting->quantity) {
                inactiveOrders.insert(resting->orderId);
                q.pop_front();
            }
        }
        if (q.empty()) {
            it = askBook.erase(it);
        } else {
            ++it;
        }
    }
    
    if (order.quantity > 0) {
        orderLookup[order.orderId] = order;
        bidBook[order.price].push_back(&orderLookup[order.orderId]);
    } else {
        inactiveOrders.insert(order.orderId);
    }
    
    if (totalFillQty > 0) {
        return {totalCost / totalFillQty, totalFillQty, "BUY"};
    }
    return {0.0, 0.0, ""};
}

FillResult MatchingEngine::processLimitSell(Order& order) {
    double totalFillQty = 0.0;
    double totalCost = 0.0;
    
    auto it = bidBook.begin();
    while (it != bidBook.end() && order.quantity > 0 && it->first >= order.price) {
        auto& q = it->second;
        while (!q.empty() && order.quantity > 0) {
            Order* resting = q.front();
            if (inactiveOrders.count(resting->orderId)) {
                q.pop_front();
                continue;
            }
            
            double matchQty = std::min(order.quantity, resting->quantity - resting->filledQuantity);
            
            totalFillQty += matchQty;
            totalCost += matchQty * resting->price;
            order.quantity -= matchQty;
            resting->filledQuantity += matchQty;
            
            if (resting->filledQuantity >= resting->quantity) {
                inactiveOrders.insert(resting->orderId);
                q.pop_front();
            }
        }
        if (q.empty()) {
            it = bidBook.erase(it);
        } else {
            ++it;
        }
    }
    
    if (order.quantity > 0) {
        orderLookup[order.orderId] = order;
        askBook[order.price].push_back(&orderLookup[order.orderId]);
    } else {
        inactiveOrders.insert(order.orderId);
    }
    
    if (totalFillQty > 0) {
        return {totalCost / totalFillQty, totalFillQty, "SELL"};
    }
    return {0.0, 0.0, ""};
}

FillResult MatchingEngine::processMarketBuy(Order& order) {
    double totalFillQty = 0.0;
    double totalCost = 0.0;
    
    auto it = askBook.begin();
    while (it != askBook.end() && order.quantity > 0) {
        auto& q = it->second;
        while (!q.empty() && order.quantity > 0) {
            Order* resting = q.front();
            if (inactiveOrders.count(resting->orderId)) {
                q.pop_front();
                continue;
            }
            
            double matchQty = std::min(order.quantity, resting->quantity - resting->filledQuantity);
            
            totalFillQty += matchQty;
            totalCost += matchQty * resting->price;
            order.quantity -= matchQty;
            resting->filledQuantity += matchQty;
            
            if (resting->filledQuantity >= resting->quantity) {
                inactiveOrders.insert(resting->orderId);
                q.pop_front();
            }
        }
        if (q.empty()) {
            it = askBook.erase(it);
        } else {
            ++it;
        }
    }
    
    if (totalFillQty > 0) {
        return {totalCost / totalFillQty, totalFillQty, "BUY"};
    }
    return {0.0, 0.0, ""};
}

FillResult MatchingEngine::processMarketSell(Order& order) {
    double totalFillQty = 0.0;
    double totalCost = 0.0;
    
    auto it = bidBook.begin();
    while (it != bidBook.end() && order.quantity > 0) {
        auto& q = it->second;
        while (!q.empty() && order.quantity > 0) {
            Order* resting = q.front();
            if (inactiveOrders.count(resting->orderId)) {
                q.pop_front();
                continue;
            }
            
            double matchQty = std::min(order.quantity, resting->quantity - resting->filledQuantity);
            
            totalFillQty += matchQty;
            totalCost += matchQty * resting->price;
            order.quantity -= matchQty;
            resting->filledQuantity += matchQty;
            
            if (resting->filledQuantity >= resting->quantity) {
                inactiveOrders.insert(resting->orderId);
                q.pop_front();
            }
        }
        if (q.empty()) {
            it = bidBook.erase(it);
        } else {
            ++it;
        }
    }
    
    if (totalFillQty > 0) {
        return {totalCost / totalFillQty, totalFillQty, "SELL"};
    }
    return {0.0, 0.0, ""};
}

int main() {
    std::string line;
    MatchingEngine engine;
    while (std::getline(std::cin, line)) {
        if (line.empty()) continue;
        
        Order o = parseOrder(line);
        FillResult fill = engine.process(o);
        std::cout << fillToJson(fill) << std::endl;
    }
    return 0;
}
