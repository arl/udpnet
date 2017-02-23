package udpnet

import (
	"fmt"
	"time"
)

type FlowControlMode int

const (
	Good FlowControlMode = iota
	Bad
)

type FlowControl struct {
	mode                FlowControlMode
	penalty             time.Duration
	goodConditions      time.Duration
	penaltyReductionAcc time.Duration // penaly reduction accumulator
}

func NewFlowControl() *FlowControl {
	fc := &FlowControl{}
	fmt.Printf("flow control initialized\n")
	fc.Reset()
	return fc
}

func (fc *FlowControl) Reset() {
	fc.mode = Bad
	// TODO: AR check if those are actually seconds or miliseconds
	fc.penalty = 4 * time.Second
	fc.goodConditions = 0
	fc.penaltyReductionAcc = 0
}

// Update flow control by providing delta and roudn trip times
func (fc *FlowControl) Update(dt, rtt time.Duration) {
	const RTTThreshold = 250 * time.Millisecond
	if fc.mode == Good {
		if rtt >= RTTThreshold {
			fmt.Printf("*** dropping to bad mode ***\n")
			fc.mode = Bad
			if fc.goodConditions < 10*time.Second && fc.penalty < 60*time.Second {
				fc.penalty *= 2
				if fc.penalty > 60*time.Second {
					fc.penalty = 60 * time.Second
				}
				fmt.Printf("penalty time increased to %v\n", fc.penalty)
			}

			fc.goodConditions = 0
			fc.penaltyReductionAcc = 0
			return
		}

		fc.goodConditions += dt
		fc.penaltyReductionAcc += dt

		if fc.penaltyReductionAcc > 10*time.Second && fc.penalty > 1*time.Second {
			fc.penalty /= 2
			if fc.penalty < 1*time.Second {
				fc.penalty = 1 * time.Second
			}
			fmt.Printf("penalty time reduced to %v\n", fc.penalty)
			fc.penaltyReductionAcc = 0
		}
	}

	if fc.mode == Bad {
		if rtt <= RTTThreshold {
			fc.goodConditions += dt
		} else {
			fc.goodConditions = 0
		}

		if fc.goodConditions > fc.penalty {
			fmt.Printf("*** upgrading to good mode ***\n")
			fc.goodConditions = 0
			fc.penaltyReductionAcc = 0
			fc.mode = Good
			return
		}
	}
}

// SendRate returns the send rate corresponding to current network conditions
//
// the send rate is the number of packets to send by second
func (fc *FlowControl) SendRate() int {
	if fc.mode == Good {
		return 30
	} else {
		return 10
	}
}
