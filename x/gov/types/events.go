package types

// Governance module event types
const (
	EventTypeSubmitProposal   = "submit_proposal"
	EventTypeProposalDeposit  = "proposal_deposit"
	EventTypeProposalVote     = "proposal_vote"
	EventTypeInactiveProposal = "inactive_proposal"
	EventTypeActiveProposal   = "active_proposal"
	EventTypeCancelProposal   = "cancel_proposal"
	EventTypeQuorumCheck      = "quorum_check"

	AttributeKeyProposalResult               = "proposal_result"
	AttributeKeyOption                       = "option"
	AttributeKeyProposalID                   = "proposal_id"
	AttributeKeyProposalMessages             = "proposal_messages" // Msg type_urls in the proposal
	AttributeKeyVotingPeriodStart            = "voting_period_start"
	AttributeKeyProposalLog                  = "proposal_log"                  // log of proposal execution
	AttributeKeyProposalQuorumResult         = "proposal_quorum_result"        // quorum of proposal
	AttributeValueProposalDropped            = "proposal_dropped"              // didn't meet min deposit
	AttributeValueProposalPassed             = "proposal_passed"               // met vote quorum
	AttributeValueProposalRejected           = "proposal_rejected"             // didn't meet vote quorum
	AttributeValueExpeditedProposalRejected  = "expedited_proposal_rejected"   // didn't meet expedited vote quorum
	AttributeValueProposalFailed             = "proposal_failed"               // error on proposal handler
	AttributeValueProposalCanceled           = "proposal_canceled"             // error on proposal handler
	AttributeValueProposalQuorumMet          = "proposal_quorum_met"           // met quorum
	AttributeValueProposalQuorumNotMet       = "proposal_quorum_not_met"       // didn't meet quorum
	AttributeValueProposalQuorumCheckSkipped = "proposal_quorum_check_skipped" // skipped quorum check

	AttributeKeyProposalType   = "proposal_type"
	AttributeSignalTitle       = "signal_title"
	AttributeSignalDescription = "signal_description"
)
